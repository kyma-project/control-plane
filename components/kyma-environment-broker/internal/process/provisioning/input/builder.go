package input

import (
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
		ApplyParameters(input *gqlschema.ClusterConfigInput, params internal.ProvisioningParametersDTO)
	}

	CreatorForPlan interface {
		IsPlanSupport(planID string) bool
		Create(parameters internal.ProvisioningParameters) (internal.ProvisionInputCreator, error)
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
}

func NewInputBuilderFactory(optComponentsSvc OptionalComponentService, disabledComponentsProvider DisabledComponentsProvider, componentsListProvider ComponentListProvider, config Config,
	defaultKymaVersion string) (CreatorForPlan, error) {

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

func (f *InputBuilderFactory) Create(pp internal.ProvisioningParameters) (internal.ProvisionInputCreator, error) {
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

	initInput, err := f.initInput(provider, pp.Parameters.KymaVersion)
	if err != nil {
		return nil, errors.Wrap(err, "while initialization input")
	}

	disabledForPlan, err := f.disabledComponentsProvider.DisabledComponentsPerPlan(pp.PlanID)
	if err != nil {
		return nil, errors.Wrap(err, "every supported plan should be specified in the disabled components map")
	}
	disabledComponents := mergeMaps(disabledForPlan, f.disabledComponentsProvider.DisabledForAll())

	return &RuntimeInput{
		input:                     initInput,
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
		return &cloudProvider.AzureTrialInput{}
	}

	switch *provider {
	case internal.Gcp:
		return &cloudProvider.GcpTrialInput{}
	default:
		return &cloudProvider.AzureTrialInput{}
	}

}
func (f *InputBuilderFactory) initInput(provider HyperscalerInputProvider, kymaVersion string) (gqlschema.ProvisionRuntimeInput, error) {
	var (
		version    string
		components internal.ComponentConfigurationInputList
	)

	if kymaVersion != "" {
		allComponents, err := f.componentsProvider.AllComponents(kymaVersion)
		if err != nil {
			return gqlschema.ProvisionRuntimeInput{}, errors.Wrapf(err, "while fetching components for %s Kyma version", kymaVersion)
		}
		version = kymaVersion
		components = mapToGQLComponentConfigurationInput(allComponents)
	} else {
		version = f.kymaVersion
		components = f.fullComponentsList
	}

	provisionInput := gqlschema.ProvisionRuntimeInput{
		RuntimeInput:  &gqlschema.RuntimeInput{},
		ClusterConfig: provider.Defaults(),
		KymaConfig: &gqlschema.KymaConfigInput{
			Version:    version,
			Components: components.DeepCopy(),
		},
	}

	provisionInput.ClusterConfig.GardenerConfig.KubernetesVersion = f.config.KubernetesVersion
	provisionInput.ClusterConfig.GardenerConfig.Purpose = &f.config.DefaultGardenerShootPurpose
	provisionInput.ClusterConfig.GardenerConfig.MachineImage = &f.config.MachineImage
	provisionInput.ClusterConfig.GardenerConfig.MachineImageVersion = &f.config.MachineImageVersion

	return provisionInput, nil
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
