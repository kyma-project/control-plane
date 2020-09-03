package input

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
	"github.com/vburenin/nsync"
)

type UpgradeKymaInput struct {
	input           gqlschema.UpgradeRuntimeInput
	mutex           *nsync.NamedMutex

	overrides       map[string][]*gqlschema.ConfigEntryInput
	globalOverrides []*gqlschema.ConfigEntryInput

	optionalComponentsService OptionalComponentService
	provisioningParameters    internal.ProvisioningParametersDTO

	componentsDisabler        ComponentsDisabler
	enabledOptionalComponents map[string]struct{}
}

func (u *UpgradeKymaInput) EnableOptionalComponent(componentName string) internal.UpgradeKymaInputCreator {
	u.mutex.Lock("enabledOptionalComponents")
	defer u.mutex.Unlock("enabledOptionalComponents")
	u.enabledOptionalComponents[componentName] = struct{}{}
	return u
}

func (u *UpgradeKymaInput) SetProvisioningParameters(params internal.ProvisioningParametersDTO) internal.UpgradeKymaInputCreator {
	u.provisioningParameters = params
	return u
}

// AppendOverrides appends overrides for the given components, the existing overrides are preserved.
func (u *UpgradeKymaInput) AppendOverrides(component string, overrides []*gqlschema.ConfigEntryInput) internal.UpgradeKymaInputCreator {
	u.mutex.Lock("AppendOverrides")
	defer u.mutex.Unlock("AppendOverrides")

	u.overrides[component] = append(u.overrides[component], overrides...)
	return u
}

// AppendAppendGlobalOverrides appends overrides, the existing overrides are preserved.
func (u *UpgradeKymaInput) AppendGlobalOverrides(overrides []*gqlschema.ConfigEntryInput) internal.UpgradeKymaInputCreator {
	u.mutex.Lock("AppendGlobalOverrides")
	defer u.mutex.Unlock("AppendGlobalOverrides")

	u.globalOverrides = append(u.globalOverrides, overrides...)
	return u
}

func (u *UpgradeKymaInput) Create() (gqlschema.UpgradeRuntimeInput, error) {
	for _, step := range []struct {
		name    string
		execute func() error
	}{
		{
			name:    "disabling components",
			execute: u.disableComponents,
		},
		{
			name:    "disabling optional components that were not selected",
			execute: u.resolveOptionalComponents,
		},
		{
			name:    "applying components overrides",
			execute: u.applyOverrides,
		},
		{
			name:    "applying global overrides",
			execute: u.applyGlobalOverrides,
		},
	} {
		if err := step.execute(); err != nil {
			return gqlschema.UpgradeRuntimeInput{}, errors.Wrapf(err, "while %s", step.name)
		}
	}

	return u.input, nil
}

func (u *UpgradeKymaInput) disableComponents() error {
	filterOut, err := u.componentsDisabler.DisableComponents(u.input.KymaConfig.Components)
	if err != nil {
		return err
	}

	u.input.KymaConfig.Components = filterOut

	return nil
}

func (u *UpgradeKymaInput) resolveOptionalComponents() error {
	u.mutex.Lock("enabledOptionalComponents")
	defer u.mutex.Unlock("enabledOptionalComponents")

	componentsToInstall := []string{}
	componentsToInstall = append(componentsToInstall, u.provisioningParameters.OptionalComponentsToInstall...)
	for name := range u.enabledOptionalComponents {
		componentsToInstall = append(componentsToInstall, name)
	}
	toDisable := u.optionalComponentsService.ComputeComponentsToDisable(componentsToInstall)

	filterOut, err := u.optionalComponentsService.ExecuteDisablers(u.input.KymaConfig.Components, toDisable...)
	if err != nil {
		return errors.Wrapf(err, "while disabling components %v", toDisable)
	}

	u.input.KymaConfig.Components = filterOut

	return nil
}

func (u *UpgradeKymaInput) applyOverrides() error {
	for i := range u.input.KymaConfig.Components {
		if entry, found := u.overrides[u.input.KymaConfig.Components[i].Component]; found {
			u.input.KymaConfig.Components[i].Configuration = append(u.input.KymaConfig.Components[i].Configuration, entry...)
		}
	}

	return nil
}

func (u *UpgradeKymaInput) applyGlobalOverrides() error {
	u.input.KymaConfig.Configuration = u.globalOverrides
	return nil
}
