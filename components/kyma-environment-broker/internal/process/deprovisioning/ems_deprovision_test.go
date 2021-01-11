package deprovisioning

import (
	"testing"

	"github.com/Peripli/service-manager-cli/pkg/types"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmsDeprovisionStep_Run(t *testing.T) {
	// given
	repo := storage.NewMemoryStorage().Operations()
	step := NewEmsDeprovisionStep(repo)
	clientFactory := servicemanager.NewFakeServiceManagerClientFactory([]types.ServiceOffering{}, []types.ServicePlan{})

	operation := internal.DeprovisioningOperation{
		Operation: internal.Operation{
			InstanceDetails: internal.InstanceDetails{
				Ems: internal.EmsData{
					Instance: internal.ServiceManagerInstanceInfo{
						BrokerID:    "broker-id",
						ServiceID:   "svc-id",
						PlanID:      "plan-id",
						InstanceID:  "instance-id",
						Provisioned: true,
					},
					BindingID: "",
					Overrides: "",
				},
			},
		},
		SMClientFactory: clientFactory,
	}
	repo.InsertDeprovisioningOperation(operation)

	// when
	operation, retry, err := step.Run(operation, logger.NewLogDummy())

	// then
	require.NoError(t, err)
	assert.Zero(t, retry)
	assert.Empty(t, operation.Ems.Instance.InstanceID)
	assert.False(t, operation.Ems.Instance.Provisioned)
	assert.False(t, operation.Ems.Instance.ProvisioningTriggered)
	clientFactory.AssertDeprovisionCalled(t, servicemanager.InstanceKey{
		BrokerID:   "broker-id",
		InstanceID: "instance-id",
		ServiceID:  "svc-id",
		PlanID:     "plan-id",
	})
}
