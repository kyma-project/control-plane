package input

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
	"github.com/vburenin/nsync"
)

type Config struct {
	URL                         string
	Timeout                     time.Duration `envconfig:"default=12h"`
	KubernetesVersion           string        `envconfig:"default=1.16.9"`
	DefaultGardenerShootPurpose string        `envconfig:"default=development"`
	MachineImage                string        `envconfig:"optional"`
	MachineImageVersion         string        `envconfig:"optional"`
}

type RuntimeInput struct {
	provisionRuntimeInput gqlschema.ProvisionRuntimeInput
	upgradeRuntimeInput   gqlschema.UpgradeRuntimeInput
	mutex                 *nsync.NamedMutex
	overrides             map[string][]*gqlschema.ConfigEntryInput
	labels                map[string]string
	globalOverrides       []*gqlschema.ConfigEntryInput

	hyperscalerInputProvider  HyperscalerInputProvider
	optionalComponentsService OptionalComponentService
	provisioningParameters    internal.ProvisioningParameters

	componentsDisabler        ComponentsDisabler
	enabledOptionalComponents map[string]struct{}
}

func (r *RuntimeInput) EnableOptionalComponent(componentName string) internal.ProvisionerInputCreator {
	r.mutex.Lock("enabledOptionalComponents")
	defer r.mutex.Unlock("enabledOptionalComponents")
	r.enabledOptionalComponents[componentName] = struct{}{}
	return r
}

func (r *RuntimeInput) SetProvisioningParameters(params internal.ProvisioningParameters) internal.ProvisionerInputCreator {
	r.provisioningParameters = params
	return r
}

// AppendOverrides sets the overrides for the given component and discard the previous ones.
//
// Deprecated: use AppendOverrides
func (r *RuntimeInput) SetOverrides(component string, overrides []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator {
	// currently same as in AppendOverrides function, as we working on the same underlying object.
	r.mutex.Lock("AppendOverrides")
	defer r.mutex.Unlock("AppendOverrides")

	r.overrides[component] = overrides
	return r
}

// AppendOverrides appends overrides for the given components, the existing overrides are preserved.
func (r *RuntimeInput) AppendOverrides(component string, overrides []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator {
	r.mutex.Lock("AppendOverrides")
	defer r.mutex.Unlock("AppendOverrides")

	r.overrides[component] = append(r.overrides[component], overrides...)
	return r
}

// AppendAppendGlobalOverrides appends overrides, the existing overrides are preserved.
func (r *RuntimeInput) AppendGlobalOverrides(overrides []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator {
	r.mutex.Lock("AppendGlobalOverrides")
	defer r.mutex.Unlock("AppendGlobalOverrides")

	r.globalOverrides = append(r.globalOverrides, overrides...)
	return r
}

func (r *RuntimeInput) SetLabel(key, value string) internal.ProvisionerInputCreator {
	r.mutex.Lock("Labels")
	defer r.mutex.Unlock("Labels")

	if r.provisionRuntimeInput.RuntimeInput.Labels == nil {
		r.provisionRuntimeInput.RuntimeInput.Labels = &gqlschema.Labels{}
	}

	(*r.provisionRuntimeInput.RuntimeInput.Labels)[key] = value
	return r
}

func (r *RuntimeInput) CreateProvisionRuntimeInput() (gqlschema.ProvisionRuntimeInput, error) {
	for _, step := range []struct {
		name    string
		execute func() error
	}{
		{
			name:    "applying provisioning parameters customization",
			execute: r.applyProvisioningParameters,
		},
		{
			name:    "disabling components",
			execute: r.disableComponentsForProvisionRuntime,
		},
		{
			name:    "disabling optional components that were not selected",
			execute: r.resolveOptionalComponentsForProvisionRuntime,
		},
		{
			name:    "applying components overrides",
			execute: r.applyOverridesForProvisionRuntime,
		},
		{
			name:    "applying global overrides",
			execute: r.applyGlobalOverridesForProvisionRuntime,
		},
	} {
		if err := step.execute(); err != nil {
			return gqlschema.ProvisionRuntimeInput{}, errors.Wrapf(err, "while %s", step.name)
		}
	}

	return r.provisionRuntimeInput, nil
}

func (r *RuntimeInput) CreateUpgradeRuntimeInput() (gqlschema.UpgradeRuntimeInput, error) {
	for _, step := range []struct {
		name    string
		execute func() error
	}{
		{
			name:    "disabling components",
			execute: r.disableComponentsForUpgradeRuntime,
		},
		{
			name:    "disabling optional components that were not selected",
			execute: r.resolveOptionalComponentsForUpgradeRuntime,
		},
		{
			name:    "applying components overrides",
			execute: r.applyOverridesForUpgradeRuntime,
		},
		{
			name:    "applying global overrides",
			execute: r.applyGlobalOverridesForUpgradeRuntime,
		},
	} {
		if err := step.execute(); err != nil {
			return gqlschema.UpgradeRuntimeInput{}, errors.Wrapf(err, "while %s", step.name)
		}
	}

	return r.upgradeRuntimeInput, nil
}

func (r *RuntimeInput) applyProvisioningParameters() error {
	params := r.provisioningParameters.Parameters
	updateString(&r.provisionRuntimeInput.RuntimeInput.Name, &params.Name)

	updateInt(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.MaxUnavailable, params.MaxUnavailable)
	updateInt(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.MaxSurge, params.MaxSurge)
	updateInt(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.AutoScalerMin, params.AutoScalerMin)
	updateInt(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.AutoScalerMax, params.AutoScalerMax)
	updateInt(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.VolumeSizeGb, params.VolumeSizeGb)
	updateString(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.Region, params.Region)
	updateString(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.MachineType, params.MachineType)
	updateString(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.TargetSecret, params.TargetSecret)
	updateString(r.provisionRuntimeInput.ClusterConfig.GardenerConfig.Purpose, params.Purpose)
	if params.LicenceType != nil {
		r.provisionRuntimeInput.ClusterConfig.GardenerConfig.LicenceType = params.LicenceType
	}

	r.hyperscalerInputProvider.ApplyParameters(r.provisionRuntimeInput.ClusterConfig, r.provisioningParameters)

	return nil
}

func (r *RuntimeInput) resolveOptionalComponentsForProvisionRuntime() error {
	r.mutex.Lock("enabledOptionalComponents")
	defer r.mutex.Unlock("enabledOptionalComponents")

	componentsToInstall := []string{}
	componentsToInstall = append(componentsToInstall, r.provisioningParameters.Parameters.OptionalComponentsToInstall...)
	for name := range r.enabledOptionalComponents {
		componentsToInstall = append(componentsToInstall, name)
	}
	toDisable := r.optionalComponentsService.ComputeComponentsToDisable(componentsToInstall)

	filterOut, err := r.optionalComponentsService.ExecuteDisablers(r.provisionRuntimeInput.KymaConfig.Components, toDisable...)
	if err != nil {
		return errors.Wrapf(err, "while disabling components %v", toDisable)
	}

	r.provisionRuntimeInput.KymaConfig.Components = filterOut

	return nil
}

func (r *RuntimeInput) resolveOptionalComponentsForUpgradeRuntime() error {
	r.mutex.Lock("enabledOptionalComponents")
	defer r.mutex.Unlock("enabledOptionalComponents")

	componentsToInstall := []string{}
	componentsToInstall = append(componentsToInstall, r.provisioningParameters.Parameters.OptionalComponentsToInstall...)
	for name := range r.enabledOptionalComponents {
		componentsToInstall = append(componentsToInstall, name)
	}
	toDisable := r.optionalComponentsService.ComputeComponentsToDisable(componentsToInstall)

	filterOut, err := r.optionalComponentsService.ExecuteDisablers(r.upgradeRuntimeInput.KymaConfig.Components, toDisable...)
	if err != nil {
		return errors.Wrapf(err, "while disabling components %v", toDisable)
	}

	r.upgradeRuntimeInput.KymaConfig.Components = filterOut

	return nil
}

func (r *RuntimeInput) disableComponentsForProvisionRuntime() error {
	filterOut, err := r.componentsDisabler.DisableComponents(r.provisionRuntimeInput.KymaConfig.Components)
	if err != nil {
		return err
	}

	r.provisionRuntimeInput.KymaConfig.Components = filterOut

	return nil
}

func (r *RuntimeInput) disableComponentsForUpgradeRuntime() error {
	filterOut, err := r.componentsDisabler.DisableComponents(r.upgradeRuntimeInput.KymaConfig.Components)
	if err != nil {
		return err
	}

	r.upgradeRuntimeInput.KymaConfig.Components = filterOut

	return nil
}

func (r *RuntimeInput) applyOverridesForProvisionRuntime() error {
	for i := range r.provisionRuntimeInput.KymaConfig.Components {
		if entry, found := r.overrides[r.provisionRuntimeInput.KymaConfig.Components[i].Component]; found {
			r.provisionRuntimeInput.KymaConfig.Components[i].Configuration = append(r.provisionRuntimeInput.KymaConfig.Components[i].Configuration, entry...)
		}
	}

	return nil
}

func (r *RuntimeInput) applyOverridesForUpgradeRuntime() error {
	for i := range r.upgradeRuntimeInput.KymaConfig.Components {
		if entry, found := r.overrides[r.upgradeRuntimeInput.KymaConfig.Components[i].Component]; found {
			r.upgradeRuntimeInput.KymaConfig.Components[i].Configuration = append(r.upgradeRuntimeInput.KymaConfig.Components[i].Configuration, entry...)
		}
	}

	return nil
}

func (r *RuntimeInput) applyGlobalOverridesForProvisionRuntime() error {
	r.provisionRuntimeInput.KymaConfig.Configuration = r.globalOverrides
	return nil
}

func (r *RuntimeInput) applyGlobalOverridesForUpgradeRuntime() error {
	r.upgradeRuntimeInput.KymaConfig.Configuration = r.globalOverrides
	return nil
}

func updateString(toUpdate *string, value *string) {
	if value != nil {
		*toUpdate = *value
	}
}

func updateInt(toUpdate *int, value *int) {
	if value != nil {
		*toUpdate = *value
	}
}
