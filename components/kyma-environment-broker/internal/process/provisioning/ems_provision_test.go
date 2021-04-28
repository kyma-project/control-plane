package provisioning

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Peripli/service-manager-cli/pkg/types"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"

	//"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/assert"
)

func TestEmsProvisioningStep_Run(t *testing.T) {
	// given
	repo := storage.NewMemoryStorage().Operations()
	clientFactory := servicemanager.NewFakeServiceManagerClientFactory([]types.ServiceOffering{}, []types.ServicePlan{})
	clientFactory.SynchronousProvisioning()
	operation := internal.ProvisioningOperation{
		Operation: internal.Operation{
			InstanceDetails: internal.InstanceDetails{
				Ems: internal.EmsData{Instance: internal.ServiceManagerInstanceInfo{
					BrokerID:  "broker-id",
					ServiceID: "svc-id",
					PlanID:    "plan-id",
				}},
				ShootDomain: "ems-test.sap.com",
			},
		},
		SMClientFactory: clientFactory,
	}
	offeringStep := NewServiceManagerOfferingStep("EMS_Offering",
		EmsOfferingName, EmsPlanName, func(op *internal.ProvisioningOperation) *internal.ServiceManagerInstanceInfo {
			return &op.Ems.Instance
		}, repo)

	provisionStep := NewEmsProvisionStep(repo)
	repo.InsertProvisioningOperation(operation)

	log := logger.NewLogDummy()
	// when
	operation, retry, err := offeringStep.Run(operation, log)
	require.NoError(t, err)
	require.Zero(t, retry)

	operation, retry, err = provisionStep.Run(operation, logger.NewLogDummy())

	// then
	assert.NoError(t, err)
	assert.Zero(t, retry)
	assert.NotEmpty(t, operation.Ems.Instance.InstanceID)
	assert.False(t, operation.Ems.Instance.Provisioned)
	assert.True(t, operation.Ems.Instance.ProvisioningTriggered)
	clientFactory.AssertProvisionCalled(t, servicemanager.InstanceKey{
		BrokerID:   "broker-id",
		InstanceID: operation.Ems.Instance.InstanceID,
		ServiceID:  "svc-id",
		PlanID:     "plan-id",
	})
}

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
}

func (c *simpleInputCreator) EnableOptionalComponent(name string) internal.ProvisionerInputCreator {
	c.enabledComponents = append(c.enabledComponents, name)
	return c
}

func (c *simpleInputCreator) SetLabel(key, val string) internal.ProvisionerInputCreator {
	c.labels[key] = val
	return c
}

func (c *simpleInputCreator) SetShootName(name string) internal.ProvisionerInputCreator {
	c.shootName = &name
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
