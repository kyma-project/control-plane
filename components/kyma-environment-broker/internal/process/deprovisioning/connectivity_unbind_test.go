package deprovisioning

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Peripli/service-manager-cli/pkg/types"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestConnectivityUnbindStep_Run(t *testing.T) {
	// given
	repo := storage.NewMemoryStorage().Operations()
	step := NewConnectivityUnbindStep(repo)
	clientFactory := servicemanager.NewFakeServiceManagerClientFactory([]types.ServiceOffering{}, []types.ServicePlan{})

	operation := internal.DeprovisioningOperation{
		Operation: internal.Operation{
			InstanceDetails: internal.InstanceDetails{
				Connectivity: internal.ConnectivityData{
					Instance: internal.ServiceManagerInstanceInfo{
						BrokerID:    "broker-id",
						ServiceID:   "svc-id",
						PlanID:      "plan-id",
						InstanceID:  "instance-id",
						Provisioned: true,
					},
					BindingID: "binding-id",
					Overrides: "eventingOverrides",
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
	assert.Empty(t, operation.Connectivity.BindingID)
	assert.Empty(t, operation.Connectivity.Overrides)
	clientFactory.AssertUnbindCalled(t, servicemanager.InstanceKey{
		BrokerID:   "broker-id",
		InstanceID: "instance-id",
		ServiceID:  "svc-id",
		PlanID:     "plan-id",
	}, "binding-id")
}
