package provisioning_test

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/Peripli/service-manager-cli/pkg/types"


	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestClsOfferingStep_Run(t *testing.T) {
	// given
	repo := storage.NewMemoryStorage().Operations()
	step := provisioning.NewClsOfferingStep(repo)
	operation := internal.ProvisioningOperation{
		Operation: internal.Operation{
			ProvisioningParameters: internal.ProvisioningParameters{},
		},
		// TODO: Change here when we move to different instantiation
		SMClientFactory: servicemanager.NewFakeServiceManagerClientFactory([]types.ServiceOffering{{
			ID:        "id-001",
			Name:      "cloud-logging",
			CatalogID: "off-cat-id",
			BrokerID:  "off-br-id",
		}}, []types.ServicePlan{{
			ID:        "plan-id",
			Name:      "standard",
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
	assert.Equal(t, "plan-cat-id", op.Cls.Instance.PlanID)
	assert.Equal(t, "off-cat-id", op.Cls.Instance.ServiceID)
	assert.Equal(t, "off-br-id", op.Cls.Instance.BrokerID)
	storedOp, _ := repo.GetProvisioningOperationByID(op.ID)
	assert.Equal(t, "plan-cat-id", storedOp.Cls.Instance.PlanID)
	assert.Equal(t, "off-cat-id", storedOp.Cls.Instance.ServiceID)
	assert.Equal(t, "off-br-id", storedOp.Cls.Instance.BrokerID)
}