/*  // +build sm_integration */

package upgrade_kyma

import (
	"fmt"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// TestEmsSteps tests all EMS steps with real Service Manager
// Usage:
// export SM_USERNAME=
// export SM_PASSWORD=
// export SM_URL=
// go test -v -tags=sm_integration ./internal/process/provisioning/... -run TestEmsSteps -count=1
const (
	secretKey = "1234567890123456"
)

func TestEmsUpgradeProvisioningSteps(t *testing.T) {
	repo := storage.NewMemoryStorage().Operations()
	cliFactory := servicemanager.NewClientFactory(servicemanager.Config{
		OverrideMode: servicemanager.SMOverrideModeNever,
		URL:          "",
		Password:     "",
		Username:     "",
	})

	offeringStep := NewServiceManagerOfferingStep("EMS_Offering",
		provisioning.EmsOfferingName, provisioning.EmsPlanName, func(op *internal.UpgradeKymaOperation) *internal.ServiceManagerInstanceInfo {
			return &op.Ems.Instance
		}, repo)

	provisioningStep := NewEmsUpgradeProvisionStep(repo)
	pp := internal.ProvisioningParameters{
		ErsContext: internal.ERSContext{
			ServiceManager: &internal.ServiceManagerEntryDTO{
				URL: os.Getenv("SM_URL"),
				Credentials: internal.ServiceManagerCredentials{
					BasicAuth: internal.ServiceManagerBasicAuth{
						Username: os.Getenv("SM_USERNAME"),
						Password: os.Getenv("SM_PASSWORD"),
					},
				},
			},
		},
	}
	operation := internal.UpgradeKymaOperation{
		Operation:       internal.Operation{ProvisioningParameters: pp},
		SMClientFactory: cliFactory,
	}
	simpleInputCreator := newInputCreator()
	operation.InputCreator = simpleInputCreator

	repo.InsertUpgradeKymaOperation(operation)

	bindingStep := NewEmsUpgradeBindStep(repo, secretKey)

	log := logrus.New()

	operation, retry, err := offeringStep.Run(operation, log)
	fmt.Printf(">>> %#v\n", operation.Ems)
	require.NoError(t, err)
	require.Zero(t, retry)

	operation, retry, err = provisioningStep.Run(operation, log)
	fmt.Printf(">>> %#v\n", operation.Ems)
	require.NoError(t, err)
	require.Zero(t, retry)

	operation, retry, err = bindingStep.Run(operation, log)
	fmt.Printf(">>> %#v\n", operation.Ems)
	require.NoError(t, err)
	//	require.Zero(t, retry)

	for i := 0; i < 30; i++ { //wait 5 min
		time.Sleep(retry)
		operation, retry, err = bindingStep.Run(operation, log)
		fmt.Printf(">>> %#v\n", operation.Ems)
		require.NoError(t, err)
		if operation.Ems.BindingID != "" {
			break
		}
	}
	require.NoError(t, err)
	require.Zero(t, retry)
	require.NotEmpty(t, operation.Ems.Instance.InstanceID)
	require.NotEmpty(t, operation.Ems.BindingID)

	overridesOut, err := decryptOverrides(secretKey, operation.Ems.Overrides)
	require.NoError(t, err)

	fmt.Printf("\nexport INSTANCE_ID=%s\nexport BINDING_ID=%s\n", operation.Ems.Instance.InstanceID, operation.Ems.BindingID)
	fmt.Printf("\nexport OVERRIDES=%#v\n", overridesOut)
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
