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
