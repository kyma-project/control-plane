package provisioning

import (
	"testing"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newInputCreator() *simpleInputCreator {
	return &simpleInputCreator{
		overrides:         make(map[string][]*gqlschema.ConfigEntryInput, 0),
		labels:            make(map[string]string),
		enabledComponents: []string{},
	}
}

type simpleInputCreator struct {
	overrides         map[string][]*gqlschema.ConfigEntryInput
	labels            map[string]string
	enabledComponents []string
	shootName         *string
	provider          internal.CloudProvider
	shootDomain       string
	shootDnsProviders gardener.DNSProvidersData
}

func (c *simpleInputCreator) EnableOptionalComponent(name string) internal.ProvisionerInputCreator {
	c.enabledComponents = append(c.enabledComponents, name)
	return c
}

func (c *simpleInputCreator) DisableOptionalComponent(name string) internal.ProvisionerInputCreator {
	for i, cmp := range c.enabledComponents {
		if cmp == name {
			c.enabledComponents = append(c.enabledComponents[:i], c.enabledComponents[i+1:]...)
		}
	}
	return c
}

func (c *simpleInputCreator) Provider() internal.CloudProvider {
	if c.provider != "" {
		return c.provider
	}
	return internal.Azure
}

func (c *simpleInputCreator) SetLabel(key, val string) internal.ProvisionerInputCreator {
	c.labels[key] = val
	return c
}

func (c *simpleInputCreator) SetShootName(name string) internal.ProvisionerInputCreator {
	c.shootName = &name
	return c
}

func (c *simpleInputCreator) SetShootDomain(name string) internal.ProvisionerInputCreator {
	c.shootDomain = name
	return c
}

func (c *simpleInputCreator) SetShootDNSProviders(providers gardener.DNSProvidersData) internal.ProvisionerInputCreator {
	c.shootDnsProviders = providers
	return c
}

func (c *simpleInputCreator) SetOverrides(component string, overrides []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator {
	return c
}

func (c *simpleInputCreator) CreateProvisionRuntimeInput() (gqlschema.ProvisionRuntimeInput, error) {
	return gqlschema.ProvisionRuntimeInput{}, nil
}

func (c *simpleInputCreator) CreateUpgradeRuntimeInput() (gqlschema.UpgradeRuntimeInput, error) {
	return gqlschema.UpgradeRuntimeInput{}, nil
}

func (c *simpleInputCreator) CreateUpgradeShootInput() (gqlschema.UpgradeShootInput, error) {
	return gqlschema.UpgradeShootInput{}, nil
}

func (c *simpleInputCreator) SetProvisioningParameters(params internal.ProvisioningParameters) internal.ProvisionerInputCreator {
	return c
}

func (c *simpleInputCreator) SetKubeconfig(_ string) internal.ProvisionerInputCreator {
	return c
}

func (c *simpleInputCreator) SetRuntimeID(_ string) internal.ProvisionerInputCreator {
	return c
}

func (c *simpleInputCreator) SetInstanceID(_ string) internal.ProvisionerInputCreator {
	return c
}

func (c *simpleInputCreator) SetClusterName(_ string) internal.ProvisionerInputCreator {
	return c
}

func (c *simpleInputCreator) SetOIDCLastValues(oidcConfig gqlschema.OIDCConfigInput) internal.ProvisionerInputCreator {
	return c
}

func (c *simpleInputCreator) AppendOverrides(component string, overrides []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator {
	c.overrides[component] = append(c.overrides[component], overrides...)
	return c
}

func (c *simpleInputCreator) AppendGlobalOverrides(overrides []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator {
	return c
}

func (c *simpleInputCreator) AssertOverride(t *testing.T, component string, cei gqlschema.ConfigEntryInput) {
	cmpOverrides, found := c.overrides[component]
	require.True(t, found)

	for _, item := range cmpOverrides {
		if item.Key == cei.Key {
			assert.Equal(t, cei, *item)
			return
		}
	}
	assert.Failf(t, "Overrides assert failed", "Expected component override not found: %+v", cei)
}

func (c *simpleInputCreator) AssertNoOverrides(t *testing.T) {
	assert.Empty(t, c.overrides)
}

func (c *simpleInputCreator) AssertLabel(t *testing.T, key, expectedValue string) {
	value, found := c.labels[key]
	require.True(t, found)
	assert.Equal(t, expectedValue, value)
}

func (c *simpleInputCreator) AssertEnabledComponent(t *testing.T, componentName string) {
	assert.Contains(t, c.enabledComponents, componentName)
}

func (c *simpleInputCreator) CreateClusterConfiguration() (reconcilerApi.Cluster, error) {
	return reconcilerApi.Cluster{}, nil
}

func (c *simpleInputCreator) CreateProvisionClusterInput() (gqlschema.ProvisionRuntimeInput, error) {
	return gqlschema.ProvisionRuntimeInput{}, nil
}
