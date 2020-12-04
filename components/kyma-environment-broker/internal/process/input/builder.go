package input

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	cloudProvider "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
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
	}

	CreatorForPlan interface {
		IsPlanSupport(planID string) bool
		CreateProvisionInput(parameters internal.ProvisioningParameters, version internal.RuntimeVersionData) (internal.ProvisionerInputCreator, error)
		CreateUpgradeInput(parameters internal.ProvisioningParameters, version internal.RuntimeVersionData) (internal.ProvisionerInputCreator, error)
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
}

func NewInputBuilderFactory(optComponentsSvc OptionalComponentService, disabledComponentsProvider DisabledComponentsProvider, componentsListProvider ComponentListProvider, config Config,
	defaultKymaVersion string, trialPlatformRegionMapping map[string]string) (CreatorForPlan, error) {

	components, err := componentsListProvider.AllComponents(defaultKymaVersion)
	if err != nil {
		return &InputBuilderFactory{}, errors.Wrap(err, "while creating components list for default Kyma version")
	}

	return &InputBuilderFactory{
		config:                     config,
		kymaVersion:                defaultKymaVersion,
		optComponentsSvc:           optComponentsSvc,
		fullComponentsList:         mapToGQLComponentConfigurationInput(components),
		componentsProvider:         componentsListProvider,
		disabledComponentsProvider: disabledComponentsProvider,
		trialPlatformRegionMapping: trialPlatformRegionMapping,
	}, nil
}

func (f *InputBuilderFactory) IsPlanSupport(planID string) bool {
	switch planID {
	case broker.GCPPlanID, broker.AzurePlanID, broker.AzureLitePlanID, broker.TrialPlanID:
		return true
	default:
		return false
	}
}

func (f *InputBuilderFactory) CreateProvisionInput(pp internal.ProvisioningParameters, version internal.RuntimeVersionData) (internal.ProvisionerInputCreator, error) {
	if !f.IsPlanSupport(pp.PlanID) {
		return nil, errors.Errorf("plan %s in not supported", pp.PlanID)
	}

	var provider HyperscalerInputProvider
	switch pp.PlanID {
	case broker.GCPPlanID:
		provider = &cloudProvider.GcpInput{}
	case broker.AzurePlanID:
		provider = &cloudProvider.AzureInput{}
	case broker.AzureLitePlanID:
		provider = &cloudProvider.AzureLiteInput{}
	case broker.TrialPlanID:
		provider = f.forTrialPlan(pp.Parameters.Provider)
		// insert cases for other providers like AWS or GCP
	default:
		return nil, errors.Errorf("case with plan %s is not supported", pp.PlanID)
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
		overrides:                 make(map[string][]*gqlschema.ConfigEntryInput, 0),
		globalOverrides:           make([]*gqlschema.ConfigEntryInput, 0),
		labels:                    make(map[string]string),
		mutex:                     nsync.NewNamedMutex(),
		hyperscalerInputProvider:  provider,
		optionalComponentsService: f.optComponentsSvc,
		componentsDisabler:        runtime.NewDisabledComponentsService(disabledComponents),
		enabledOptionalComponents: map[string]struct{}{},
	}, nil
}

func (f *InputBuilderFactory) forTrialPlan(provider *internal.TrialCloudProvider) HyperscalerInputProvider {
	if provider == nil {
		return &cloudProvider.AzureTrialInput{
			PlatformRegionMapping: f.trialPlatformRegionMapping,
		}
	}

	switch *provider {
	case internal.Gcp:
		return &cloudProvider.GcpTrialInput{
			PlatformRegionMapping: f.trialPlatformRegionMapping,
		}
	default:
		return &cloudProvider.AzureTrialInput{
			PlatformRegionMapping: f.trialPlatformRegionMapping,
		}
	}

}
func (f *InputBuilderFactory) initProvisionRuntimeInput(provider HyperscalerInputProvider, version internal.RuntimeVersionData) (gqlschema.ProvisionRuntimeInput, error) {
	var (
		components internal.ComponentConfigurationInputList
	)

	switch version.Origin {
	case internal.Defaults:
		components = f.fullComponentsList
	case internal.Parameters, internal.GlobalAccount:
		allComponents, err := f.componentsProvider.AllComponents(version.Version)
		if err != nil {
			return gqlschema.ProvisionRuntimeInput{}, errors.Wrapf(err, "while fetching components for %s Kyma version", version.Version)
		}

		components = mapToGQLComponentConfigurationInput(allComponents)
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
	if provisionInput.ClusterConfig.GardenerConfig.Name == nil {
		provisionInput.ClusterConfig.GardenerConfig.Name = ptr.String("")
	}
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

	upgradeKymaInput, err := f.initUpgradeRuntimeInput(version)
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
	}, nil
}

func (f *InputBuilderFactory) initUpgradeRuntimeInput(version internal.RuntimeVersionData) (gqlschema.UpgradeRuntimeInput, error) {
	if version.Version == "" {
		return gqlschema.UpgradeRuntimeInput{}, errors.New("desired runtiem version cannot be empty")
	}

	return gqlschema.UpgradeRuntimeInput{
		KymaConfig: &gqlschema.KymaConfigInput{
			Version:    version.Version,
			Components: f.fullComponentsList.DeepCopy(),
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
