package provisioning_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Peripli/service-manager-cli/pkg/types"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestServiceManagerOfferingStep_Run(t *testing.T) {
	// given
	repo := storage.NewMemoryStorage().Operations()
	step := provisioning.NewServiceManagerOfferingStep("xsuaa-offering", "xsuaa", "application",
		xsuaaExtractor, repo)
	operation := internal.ProvisioningOperation{
		Operation: internal.Operation{
			ProvisioningParameters: internal.ProvisioningParameters{},
		},
		SMClientFactory: servicemanager.NewFakeServiceManagerClientFactory([]types.ServiceOffering{{
			ID:        "id-001",
			Name:      "xsuaa",
			CatalogID: "off-cat-id",
			BrokerID:  "off-br-id",
		}}, []types.ServicePlan{{
			ID:        "plan-id",
			Name:      "application",
			CatalogID: "plan-cat-id",
		},
		}),
	}
	err := repo.InsertProvisioningOperation(operation)
	require.NoError(t, err)

	// when
	op, retry, err := step.Run(operation, logger.NewLogDummy())

	// then
	assert.Zero(t, retry)
	assert.NoError(t, err)
	assert.Equal(t, "plan-cat-id", op.XSUAA.Instance.PlanID)
	assert.Equal(t, "off-cat-id", op.XSUAA.Instance.ServiceID)
	assert.Equal(t, "off-br-id", op.XSUAA.Instance.BrokerID)
	storedOp, _ := repo.GetProvisioningOperationByID(op.ID)
	assert.Equal(t, "plan-cat-id", storedOp.XSUAA.Instance.PlanID)
	assert.Equal(t, "off-cat-id", storedOp.XSUAA.Instance.ServiceID)
	assert.Equal(t, "off-br-id", storedOp.XSUAA.Instance.BrokerID)
}

func xsuaaExtractor(op *internal.ProvisioningOperation) *internal.ServiceManagerInstanceInfo {
	return &op.XSUAA.Instance
}
