package provisioning_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Peripli/service-manager-cli/pkg/types"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	uaa "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager/xsuaa"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestXSUAAProvisioningStep_Run(t *testing.T) {
	// given
	repo := storage.NewMemoryStorage().Operations()
	step := provisioning.NewXSUAAProvisioningStep(repo, uaa.Config{
		DeveloperGroup:      "dg",
		DeveloperRole:       "dr",
		NamespaceAdminGroup: "nag",
		NamespaceAdminRole:  "nar",
	})
	clientFactory := servicemanager.NewFakeServiceManagerClientFactory([]types.ServiceOffering{}, []types.ServicePlan{})
	clientFactory.SynchronousProvisioning()
	operation := internal.ProvisioningOperation{
		Operation: internal.Operation{
			ProvisioningParameters: internal.ProvisioningParameters{},
			InstanceDetails: internal.InstanceDetails{
				XSUAA: internal.XSUAAData{Instance: internal.ServiceManagerInstanceInfo{
					BrokerID:  "broker-id",
					ServiceID: "svc-id",
					PlanID:    "plan-id",
				},
				},
				ShootDomain: "uaa-test.sap.com"},
		},
		SMClientFactory: clientFactory,
	}
	err := repo.InsertProvisioningOperation(operation)
	require.NoError(t, err)

	// when
	operation, retry, err := step.Run(operation, logger.NewLogDummy())

	// then
	assert.NoError(t, err)
	assert.Zero(t, retry)
	assert.NotEmpty(t, operation.XSUAA.Instance.InstanceID)
	assert.True(t, operation.XSUAA.Instance.Provisioned)
	assert.True(t, operation.XSUAA.Instance.ProvisioningTriggered)
	clientFactory.AssertProvisionCalled(t, servicemanager.InstanceKey{
		BrokerID:   "broker-id",
		InstanceID: operation.XSUAA.Instance.InstanceID,
		ServiceID:  "svc-id",
		PlanID:     "plan-id",
	})
}
