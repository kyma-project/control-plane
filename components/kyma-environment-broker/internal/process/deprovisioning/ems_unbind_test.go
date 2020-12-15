package deprovisioning_test

import (
	"github.com/Peripli/service-manager-cli/pkg/types"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/deprovisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEmsUnbindStep_Run(t *testing.T) {
	// given
	repo := storage.NewMemoryStorage().Operations()
	step := deprovisioning.NewEmsDeprovisionStep(repo)
	clientFactory := servicemanager.NewFakeServiceManagerClientFactory([]types.ServiceOffering{}, []types.ServicePlan{})

	operation := internal.DeprovisioningOperation{
		ProvisioningParameters: "{}",
		SMClientFactory:        clientFactory,
		Ems: internal.EmsData{
			Instance: internal.ServiceManagerInstanceInfo{
				BrokerID:   "broker-id",
				ServiceID:  "svc-id",
				PlanID:     "plan-id",
				InstanceID: "instance-id",
			},
			BindingID: "binding-id",
		},
	}
	repo.InsertDeprovisioningOperation(operation)

	// when
	operation, retry, err := step.Run(operation, logger.NewLogDummy())

	// then
	require.NoError(t, err)
	assert.Empty(t, operation.Ems.BindingID)
	assert.Zero(t, retry)
	clientFactory.AssertUnbindCalled(t, servicemanager.InstanceKey{
		BrokerID:   "broker-id",
		InstanceID: operation.Ems.Instance.InstanceID,
		ServiceID:  "svc-id",
		PlanID:     "plan-id",
	}, "binding-id")
}