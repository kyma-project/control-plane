package upgrade_kyma

import (
	"github.com/Peripli/service-manager-cli/pkg/types"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClsUpgradeOfferingStep_Run(t *testing.T) {
	// given
	clsConfig := &cls.Config{
		ServiceManager: &cls.ServiceManagerConfig{
			Credentials: []*cls.ServiceManagerCredentials{
				{
					Region:   "eu",
					URL:      "http://service-manager",
					Username: "qwerty",
					Password: "qwerty",
				},
			},
		},
	}

	repo := storage.NewMemoryStorage().Operations()
	step := NewClsUpgradeOfferingStep(clsConfig, repo)
	operation := internal.UpgradeKymaOperation{
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
	err := repo.InsertUpgradeKymaOperation(operation)
	require.NoError(t, err)

	// when
	op, retry, err := step.Run(operation, logger.NewLogDummy())

	// then
	assert.Zero(t, retry)
	assert.NoError(t, err)
	assert.Equal(t, "plan-cat-id", op.Cls.Instance.PlanID)
	assert.Equal(t, "off-cat-id", op.Cls.Instance.ServiceID)
	assert.Equal(t, "off-br-id", op.Cls.Instance.BrokerID)
	storedOp, _ := repo.GetUpgradeKymaOperationByID(op.Operation.ID)
	assert.Equal(t, "plan-cat-id", storedOp.Cls.Instance.PlanID)
	assert.Equal(t, "off-cat-id", storedOp.Cls.Instance.ServiceID)
	assert.Equal(t, "off-br-id", storedOp.Cls.Instance.BrokerID)
}
