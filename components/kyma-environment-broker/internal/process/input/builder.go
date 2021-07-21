package input

import (
	"fmt"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	cloudProvider "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/pkg/errors"
	"github.com/vburenin/nsync"
)

//go:generate mockery -name=ComponentListProvider -output=automock -outpkg=automock -case=underscore
//go:generate mockery -name=CreatorForPlan -output=automock -outpkg=automock -case=underscore
//go:generate mockery -name=ComponentsDisabler -output=automock -outpkg=automock -case=underscore

type (
	OptionalComponentService interface {
		ExecuteDisablers(components internal.ComponentConfigurationInputList, names ...string) (internal.ComponentConfigurationInputList, error)
		ComputeComponentsToDisable(optComponentsToKeep []string) []string
	}

	ComponentsDisabler interface {
		DisableComponents(components internal.ComponentConfigurationInputList) (internal.ComponentConfigurationInputList, error)
	}

	DisabledComponentsProvider interface {
		DisabledComponentsPerPlan(planID string) (map[string]struct{}, error)
		DisabledForAll() map[string]struct{}
	}

	HyperscalerInputProvider interface {
		Defaults() *gqlschema.ClusterConfigInput
		ApplyParameters(input *gqlschema.ClusterConfigInput, params internal.ProvisioningParameters)
		Profile() gqlschema.KymaProfile
		Provider() internal.CloudProvider
	}

	CreatorForPlan interface {
		IsPlanSupport(planID string) bool
		CreateProvisionInput(parameters internal.ProvisioningParameters, version internal.RuntimeVersionData) (internal.ProvisionerInputCreator, error)
		CreateUpgradeInput(parameters internal.ProvisioningParameters, version internal.RuntimeVersionData) (internal.ProvisionerInputCreator, error)
		CreateUpgradeShootInput(parameters internal.ProvisioningParameters) (internal.ProvisionerInputCreator, error)
	}

	ComponentListProvider interface {
		AllComponents(kymaVersion string) ([]v1alpha1.KymaComponent, error)
	}
)

type InputBuilderFactory struct {
	kymaVersion                string
	config                     Config
	optComponentsSvc           OptionalComponentService
	fullComponentsList         internal.ComponentConfigurationInputList
	componentsProvider         ComponentListProvider
	disabledComponentsProvider DisabledComponentsProvider
	trialPlatformRegionMapping map[string]string
	enabledFreemiumProviders   map[string]struct{}
	oidcDefaultValues          internal.OIDCConfigDTO
}

func NewInputBuilderFactory(optComponentsSvc OptionalComponentService, disabledComponentsProvider DisabledComponentsProvider, componentsListProvider ComponentListProvider, config Config,
	defaultKymaVersion string, trialPlatformRegionMapping map[string]string, enabledFreemiumProviders []string, oidcValues internal.OIDCConfigDTO) (CreatorForPlan, error) {

	components, err := componentsListProvider.AllComponents(defaultKymaVersion)
	if err != nil {
		return &InputBuilderFactory{}, errors.Wrap(err, "while creating components list for default Kyma version")
	}

	freemiumProviders := map[string]struct{}{}
	for _, p := range enabledFreemiumProviders {
		freemiumProviders[strings.ToLower(p)] = struct{}{}
	}

	return &InputBuilderFactory{
		kymaVersion:                defaultKymaVersion,
		config:                     config,
		optComponentsSvc:           optComponentsSvc,
		fullComponentsList:         mapToGQLComponentConfigurationInput(components),
		componentsProvider:         componentsListProvider,
		disabledComponentsProvider: disabledComponentsProvider,
		trialPlatformRegionMapping: trialPlatformRegionMapping,
		enabledFreemiumProviders:   freemiumProviders,
		oidcDefaultValues:          oidcValues,
	}, nil
}

func (f *InputBuilderFactory) IsPlanSupport(planID string) bool {
	switch planID {
	case broker.AWSPlanID, broker.AWSHAPlanID, broker.GCPPlanID, broker.AzurePlanID, broker.FreemiumPlanID,
		broker.AzureLitePlanID, broker.TrialPlanID, broker.OpenStackPlanID, broker.AzureHAPlanID:
		return true
	default:
		return false
	}
}

func (f *InputBuilderFactory) getHyperscalerProviderForPlanID(planID string, pp internal.ProvisioningParameters) (HyperscalerInputProvider, error) {
	var provider HyperscalerInputProvider
	switch planID {
	case broker.GCPPlanID:
		provider = &cloudProvider.GcpInput{}
	case broker.FreemiumPlanID:
		return f.forFreemiumPlan(pp)
	case broker.OpenStackPlanID:
		provider = &cloudProvider.OpenStackInput{}
	case broker.AzurePlanID:
		provider = &cloudProvider.AzureInput{}
	case broker.AzureLitePlanID:
		provider = &cloudProvider.AzureLiteInput{}
	case broker.AzureHAPlanID:
		provider = &cloudProvider.AzureHAInput{}
	case broker.TrialPlanID:
		provider = f.forTrialPlan(pp.Parameters.Provider)
	case broker.AWSPlanID:
		provider = &cloudProvider.AWSInput{}
	case broker.AWSHAPlanID:
		provider = &cloudProvider.AWSHAInput{}
		// insert cases for other providers like AWS or GCP
	default:
		return nil, errors.Errorf("case with plan %s is not supported", planID)
	}
	return provider, nil
}

func (f *InputBuilderFactory) CreateProvisionInput(pp internal.ProvisioningParameters, version internal.RuntimeVersionData) (internal.ProvisionerInputCreator, error) {
	if !f.IsPlanSupport(pp.PlanID) {
		return nil, errors.Errorf("plan %s in not supported", pp.PlanID)
	}

	provider, err := f.getHyperscalerProviderForPlanID(pp.PlanID, pp)
	if err != nil {
		return nil, errors.Wrap(err, "during createing provision input")
	}

	initInput, err := f.initProvisionRuntimeInput(provider, version)
	if err != nil {
		return nil, errors.Wrap(err, "while initializing ProvisionRuntimeInput")
	}

	disabledForPlan, err := f.disabledComponentsProvider.DisabledComponentsPerPlan(pp.PlanID)
	if err != nil {
		return nil, errors.Wrap(err, "every supported plan should be specified in the disabled components map")
	}
	disabledComponents := mergeMaps(disabledForPlan, f.disabledComponentsProvider.DisabledForAll())

	return &RuntimeInput{
		provisionRuntimeInput:     initInput,
		mutex:                     nsync.NewNamedMutex(),
		overrides:                 make(map[string][]*gqlschema.ConfigEntryInput, 0),
		labels:                    make(map[string]string),
		globalOverrides:           make([]*gqlschema.ConfigEntryInput, 0),
		hyperscalerInputProvider:  provider,
		optionalComponentsService: f.optComponentsSvc,
		provisioningParameters:    pp,
		componentsDisabler:        runtime.NewDisabledComponentsService(disabledComponents),
		enabledOptionalComponents: map[string]struct{}{},
		oidcDefaultValues:         f.oidcDefaultValues,
		trialNodesNumber:          f.config.TrialNodesNumber,
	}, nil
}

func (f *InputBuilderFactory) forTrialPlan(provider *internal.CloudProvider) HyperscalerInputProvider {
	var trialProvider internal.CloudProvider
	if provider == nil {
		trialProvider = f.config.DefaultTrialProvider
	} else {
		trialProvider = *provider
	}

	switch trialProvider {
	case internal.GCP:
		return &cloudProvider.GcpTrialInput{
			PlatformRegionMapping: f.trialPlatformRegionMapping,
		}
	case internal.AWS:
		return &cloudProvider.AWSTrialInput{
			PlatformRegionMapping: f.trialPlatformRegionMapping,
		}
	default:
		return &cloudProvider.AzureTrialInput{
			PlatformRegionMapping: f.trialPlatformRegionMapping,
		}
	}

}

func (f *InputBuilderFactory) provideComponentList(version internal.RuntimeVersionData) (internal.ComponentConfigurationInputList, error) {
	switch version.Origin {
	case internal.Defaults:
		return f.fullComponentsList, nil
	case internal.Parameters, internal.AccountMapping:
		allComponents, err := f.componentsProvider.AllComponents(version.Version)
		if err != nil {
			return internal.ComponentConfigurationInputList{}, errors.Wrapf(err, "while fetching components for %s Kyma version", version.Version)
		}
		return mapToGQLComponentConfigurationInput(allComponents), nil
	}

	return internal.ComponentConfigurationInputList{}, errors.Errorf("Unknown version.Origin: %s", version.Origin)
}

func (f *InputBuilderFactory) initProvisionRuntimeInput(provider HyperscalerInputProvider, version internal.RuntimeVersionData) (gqlschema.ProvisionRuntimeInput, error) {
	components, err := f.provideComponentList(version)
	if err != nil {
		return gqlschema.ProvisionRuntimeInput{}, err
	}
	kymaProfile := provider.Profile()

	provisionInput := gqlschema.ProvisionRuntimeInput{
		RuntimeInput:  &gqlschema.RuntimeInput{},
		ClusterConfig: provider.Defaults(),
		KymaConfig: &gqlschema.KymaConfigInput{
			Profile:    &kymaProfile,
			Version:    version.Version,
			Components: components.DeepCopy(),
		},
	}

	provisionInput.ClusterConfig.GardenerConfig.KubernetesVersion = f.config.KubernetesVersion
	if provisionInput.ClusterConfig.GardenerConfig.Purpose == nil {
		provisionInput.ClusterConfig.GardenerConfig.Purpose = &f.config.DefaultGardenerShootPurpose
	}
	if f.config.MachineImage != "" {
		provisionInput.ClusterConfig.GardenerConfig.MachineImage = &f.config.MachineImage
	}
	if f.config.MachineImageVersion != "" {
		provisionInput.ClusterConfig.GardenerConfig.MachineImageVersion = &f.config.MachineImageVersion
	}
	return provisionInput, nil
}

func (f *InputBuilderFactory) CreateUpgradeInput(pp internal.ProvisioningParameters, version internal.RuntimeVersionData) (internal.ProvisionerInputCreator, error) {
	if !f.IsPlanSupport(pp.PlanID) {
		return nil, errors.Errorf("plan %s in not supported", pp.PlanID)
	}

	provider, err := f.getHyperscalerProviderForPlanID(pp.PlanID, pp)
	if err != nil {
		return nil, errors.Wrap(err, "during createing provision input")
	}

	upgradeKymaInput, err := f.initUpgradeRuntimeInput(version, provider)
	if err != nil {
		return nil, errors.Wrap(err, "while initializing UpgradeRuntimeInput")
	}

	disabledForPlan, err := f.disabledComponentsProvider.DisabledComponentsPerPlan(pp.PlanID)
	if err != nil {
		return nil, errors.Wrap(err, "every supported plan should be specified in the disabled components map")
	}
	disabledComponents := mergeMaps(disabledForPlan, f.disabledComponentsProvider.DisabledForAll())

	return &RuntimeInput{
		upgradeRuntimeInput:       upgradeKymaInput,
		mutex:                     nsync.NewNamedMutex(),
		overrides:                 make(map[string][]*gqlschema.ConfigEntryInput, 0),
		globalOverrides:           make([]*gqlschema.ConfigEntryInput, 0),
		optionalComponentsService: f.optComponentsSvc,
		componentsDisabler:        runtime.NewDisabledComponentsService(disabledComponents),
		enabledOptionalComponents: map[string]struct{}{},
		trialNodesNumber:          f.config.TrialNodesNumber,
	}, nil
}

func (f *InputBuilderFactory) initUpgradeRuntimeInput(version internal.RuntimeVersionData, provider HyperscalerInputProvider) (gqlschema.UpgradeRuntimeInput, error) {
	if version.Version == "" {
		return gqlschema.UpgradeRuntimeInput{}, errors.New("desired runtime version cannot be empty")
	}

	kymaProfile := provider.Profile()
	components, err := f.provideComponentList(version)
	if err != nil {
		return gqlschema.UpgradeRuntimeInput{}, err
	}

	return gqlschema.UpgradeRuntimeInput{
		KymaConfig: &gqlschema.KymaConfigInput{
			Profile:    &kymaProfile,
			Version:    version.Version,
			Components: components.DeepCopy(),
		},
	}, nil
}

func mapToGQLComponentConfigurationInput(kymaComponents []v1alpha1.KymaComponent) internal.ComponentConfigurationInputList {
	var input internal.ComponentConfigurationInputList
	for _, component := range kymaComponents {
		var sourceURL *string
		if component.Source != nil {
			sourceURL = &component.Source.URL
		}

		input = append(input, &gqlschema.ComponentConfigurationInput{
			Component: component.Name,
			Namespace: component.Namespace,
			SourceURL: sourceURL,
		})
	}
	return input
}

func mergeMaps(maps ...map[string]struct{}) map[string]struct{} {
	res := map[string]struct{}{}
	for _, m := range maps {
		for k, v := range m {
			res[k] = v
		}
	}
	return res
}

func (f *InputBuilderFactory) CreateUpgradeShootInput(pp internal.ProvisioningParameters) (internal.ProvisionerInputCreator, error) {
	if !f.IsPlanSupport(pp.PlanID) {
		return nil, errors.Errorf("plan %s in not supported", pp.PlanID)
	}

	provider, err := f.getHyperscalerProviderForPlanID(pp.PlanID, pp)
	if err != nil {
		return nil, errors.Wrap(err, "during createing provision input")
	}

	input := f.initUpgradeShootInput(provider)
	return &RuntimeInput{
		upgradeShootInput:        input,
		mutex:                    nsync.NewNamedMutex(),
		hyperscalerInputProvider: provider,
		trialNodesNumber:         f.config.TrialNodesNumber,
	}, nil
}

func (f *InputBuilderFactory) initUpgradeShootInput(provider HyperscalerInputProvider) gqlschema.UpgradeShootInput {
	input := gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion: &f.config.KubernetesVersion,
		},
	}

	if f.config.MachineImage != "" {
		input.GardenerConfig.MachineImage = &f.config.MachineImage
	}
	if f.config.MachineImageVersion != "" {
		input.GardenerConfig.MachineImageVersion = &f.config.MachineImageVersion
	}

	return input
}

func (f *InputBuilderFactory) forFreemiumPlan(pp internal.ProvisioningParameters) (HyperscalerInputProvider, error) {
	provider := pp.PlatformProvider

	if !f.IsFreemiumProviderEnabled(provider) {
		return nil, fmt.Errorf("freemium provider %s is not enabled", provider)
	}
	switch provider {
	case internal.AWS:
		return &cloudProvider.AWSFreemiumInput{}, nil
	case internal.Azure:
		return &cloudProvider.AzureFreemiumInput{}, nil
	default:
		return nil, fmt.Errorf("provider %s is not supported", provider)
	}
}

func (f *InputBuilderFactory) IsFreemiumProviderEnabled(provider internal.CloudProvider) bool {
	_, found := f.enabledFreemiumProviders[strings.ToLower(string(provider))]
	return found
}
