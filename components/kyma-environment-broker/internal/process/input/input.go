package input

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
	"github.com/vburenin/nsync"
)

const (
	trialSuffixLength    = 5
	maxRuntimeNameLength = 36
)

type Config struct {
	URL                         string
	ProvisioningTimeout         time.Duration          `envconfig:"default=6h"`
	DeprovisioningTimeout       time.Duration          `envconfig:"default=5h"`
	KubernetesVersion           string                 `envconfig:"default=1.16.9"`
	DefaultGardenerShootPurpose string                 `envconfig:"default=development"`
	MachineImage                string                 `envconfig:"optional"`
	MachineImageVersion         string                 `envconfig:"optional"`
	TrialNodesNumber            int                    `envconfig:"optional"`
	DefaultTrialProvider        internal.CloudProvider `envconfig:"default=Azure"` // could be: Azure, AWS, GCP
}

type RuntimeInput struct {
	provisionRuntimeInput gqlschema.ProvisionRuntimeInput
	upgradeRuntimeInput   gqlschema.UpgradeRuntimeInput
	upgradeShootInput     gqlschema.UpgradeShootInput
	mutex                 *nsync.NamedMutex
	overrides             map[string][]*gqlschema.ConfigEntryInput
	labels                map[string]string
	globalOverrides       []*gqlschema.ConfigEntryInput

	hyperscalerInputProvider  HyperscalerInputProvider
	optionalComponentsService OptionalComponentService
	provisioningParameters    internal.ProvisioningParameters
	shootName                 *string

	componentsDisabler        ComponentsDisabler
	enabledOptionalComponents map[string]struct{}
	oidcDefaultValues         internal.OIDCConfigDTO

	trialNodesNumber int
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

func (r *RuntimeInput) SetShootName(name string) internal.ProvisionerInputCreator {
	r.shootName = &name
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

// AppendGlobalOverrides appends overrides, the existing overrides are preserved.
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
			execute: r.applyProvisioningParametersForProvisionRuntime,
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
		{
			name:    "applying global configuration",
			execute: r.applyGlobalConfigurationForProvisionRuntime,
		},
		{
			name:    "removing forbidden chars and adding random string to runtime name",
			execute: r.adjustRuntimeName,
		},
		{
			name:    "set number of nodes from configuration",
			execute: r.setNodesForTrialProvision,
		},
		{
			name:    "configure OIDC",
			execute: r.configureOIDC,
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
		{
			name:    "applying global configuration",
			execute: r.applyGlobalConfigurationForUpgradeRuntime,
		},
		{
			name:    "set number of nodes from configuration",
			execute: r.setNodesForTrialProvision,
		},
	} {
		if err := step.execute(); err != nil {
			return gqlschema.UpgradeRuntimeInput{}, errors.Wrapf(err, "while %s", step.name)
		}
	}

	return r.upgradeRuntimeInput, nil
}

func (r *RuntimeInput) CreateUpgradeShootInput() (gqlschema.UpgradeShootInput, error) {

	for _, step := range []struct {
		name    string
		execute func() error
	}{
		{
			name:    "applying provisioning parameters customization",
			execute: r.applyProvisioningParametersForUpgradeShoot,
		},
		{
			name:    "setting number of trial nodes from configuration",
			execute: r.setNodesForTrialUpgrade,
		},
		{
			name:    "configure OIDC",
			execute: r.configureOIDC,
		},
	} {
		if err := step.execute(); err != nil {
			return gqlschema.UpgradeShootInput{}, errors.Wrapf(err, "while %s", step.name)
		}
	}
	return r.upgradeShootInput, nil
}

func (r *RuntimeInput) Provider() internal.CloudProvider {
	return r.hyperscalerInputProvider.Provider()
}

func (r *RuntimeInput) applyProvisioningParametersForProvisionRuntime() error {
	params := r.provisioningParameters.Parameters
	updateString(&r.provisionRuntimeInput.RuntimeInput.Name, &params.Name)

	updateInt(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.MaxUnavailable, params.MaxUnavailable)
	updateInt(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.MaxSurge, params.MaxSurge)
	updateInt(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.AutoScalerMin, params.AutoScalerMin)
	updateInt(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.AutoScalerMax, params.AutoScalerMax)
	updateInt(r.provisionRuntimeInput.ClusterConfig.GardenerConfig.VolumeSizeGb, params.VolumeSizeGb)
	updateString(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.Name, r.shootName)
	updateString(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.Region, params.Region)
	updateString(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.MachineType, params.MachineType)
	updateString(&r.provisionRuntimeInput.ClusterConfig.GardenerConfig.TargetSecret, params.TargetSecret)
	updateString(r.provisionRuntimeInput.ClusterConfig.GardenerConfig.Purpose, params.Purpose)
	if params.LicenceType != nil {
		r.provisionRuntimeInput.ClusterConfig.GardenerConfig.LicenceType = params.LicenceType
	}
	r.provisionRuntimeInput.ClusterConfig.Administrators = []string{r.provisioningParameters.ErsContext.UserID}

	r.hyperscalerInputProvider.ApplyParameters(r.provisionRuntimeInput.ClusterConfig, r.provisioningParameters)

	return nil
}

func (r *RuntimeInput) applyProvisioningParametersForUpgradeShoot() error {
	// As of now cluster upgrade doesn't support upgrading parameters which could also be specified as provisioning parameters
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

func (r *RuntimeInput) applyGlobalConfigurationForProvisionRuntime() error {
	strategy := gqlschema.ConflictStrategyReplace
	r.provisionRuntimeInput.KymaConfig.ConflictStrategy = &strategy
	return nil
}

func (r *RuntimeInput) applyGlobalConfigurationForUpgradeRuntime() error {
	strategy := gqlschema.ConflictStrategyReplace
	r.upgradeRuntimeInput.KymaConfig.ConflictStrategy = &strategy
	return nil
}

func (r *RuntimeInput) adjustRuntimeName() error {

	reg, err := regexp.Compile("[^a-zA-Z0-9\\-\\.]+")
	if err != nil {
		return errors.Wrap(err, "while compiling regexp")
	}

	name := strings.ToLower(reg.ReplaceAllString(r.provisionRuntimeInput.RuntimeInput.Name, ""))
	modifiedLength := len(name) + trialSuffixLength + 1
	if modifiedLength > maxRuntimeNameLength {
		name = trimLastCharacters(name, modifiedLength-maxRuntimeNameLength)
	}

	r.provisionRuntimeInput.RuntimeInput.Name = fmt.Sprintf("%s-%s", name, randomString(trialSuffixLength))
	return nil
}

func (r *RuntimeInput) configureOIDC() error {
	// set default or provided params to provisioning/update inpuit (if exists)
	// This method could be used for:
	// provisioning (upgradeShootInput.GardenerConfig is nil)
	// or upgrade (provisionRuntimeInput.ClusterConfig is nil)

	oidcParamsToSet := &gqlschema.OIDCConfigInput{
		ClientID:       r.oidcDefaultValues.ClientID,
		GroupsClaim:    r.oidcDefaultValues.GroupsClaim,
		IssuerURL:      r.oidcDefaultValues.IssuerURL,
		SigningAlgs:    r.oidcDefaultValues.SigningAlgs,
		UsernameClaim:  r.oidcDefaultValues.UsernameClaim,
		UsernamePrefix: r.oidcDefaultValues.UsernamePrefix,
	}
	if r.provisioningParameters.Parameters.OIDC.IsProvided() {
		oidc := r.provisioningParameters.Parameters.OIDC
		oidcParamsToSet = &gqlschema.OIDCConfigInput{
			ClientID:       oidc.ClientID,
			GroupsClaim:    oidc.GroupsClaim,
			IssuerURL:      oidc.IssuerURL,
			SigningAlgs:    oidc.SigningAlgs,
			UsernameClaim:  oidc.UsernameClaim,
			UsernamePrefix: oidc.UsernamePrefix,
		}
	}

	if r.provisionRuntimeInput.ClusterConfig != nil {
		r.provisionRuntimeInput.ClusterConfig.GardenerConfig.OidcConfig = oidcParamsToSet
	}
	if r.upgradeShootInput.GardenerConfig != nil {
		r.upgradeShootInput.GardenerConfig.OidcConfig = oidcParamsToSet
	}
	return nil
}

func (r *RuntimeInput) setNodesForTrialProvision() error {
	// parameter with number of notes for trial plan is optional; if parameter is not set value is equal to 0
	if r.trialNodesNumber == 0 {
		return nil
	}
	if broker.IsTrialPlan(r.provisioningParameters.PlanID) {
		r.provisionRuntimeInput.ClusterConfig.GardenerConfig.AutoScalerMin = r.trialNodesNumber
		r.provisionRuntimeInput.ClusterConfig.GardenerConfig.AutoScalerMax = r.trialNodesNumber
	}
	return nil
}

func (r *RuntimeInput) setNodesForTrialUpgrade() error {
	// parameter with number of nodes for trial plan is optional; if parameter is not set value is equal to 0
	if r.trialNodesNumber == 0 {
		return nil
	}
	if broker.IsTrialPlan(r.provisioningParameters.PlanID) {
		r.upgradeShootInput.GardenerConfig.AutoScalerMin = &r.trialNodesNumber
		r.upgradeShootInput.GardenerConfig.AutoScalerMax = &r.trialNodesNumber
	}
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

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz")

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func trimLastCharacters(s string, count int) string {
	s = s[:len(s)-count]
	return s
}
