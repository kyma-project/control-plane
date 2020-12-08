package kyma_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/kyma"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const poolingInterval = 20 * time.Millisecond

func TestUpgradeKymaManager_Execute(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		// given
		store := storage.NewMemoryStorage()

		resolver := &automock.RuntimeResolver{}
		defer resolver.AssertExpectations(t)

		resolver.On("Resolve", orchestration.TargetSpec{
			Include: nil,
			Exclude: nil,
		}).Return([]orchestration.Runtime{}, nil)

		id := "id"
		err := store.Orchestrations().Insert(internal.Orchestration{OrchestrationID: id, State: orchestration.Pending})
		require.NoError(t, err)

		svc := kyma.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), nil, resolver, 20*time.Millisecond, logrus.New())

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Succeeded, o.State)
	})
	t.Run("InProgress", func(t *testing.T) {
		// given
		store := storage.NewMemoryStorage()

		resolver := &automock.RuntimeResolver{}
		defer resolver.AssertExpectations(t)

		id := "id"
		err := store.Orchestrations().Insert(internal.Orchestration{
			OrchestrationID: id,
			State:           orchestration.InProgress,
			Parameters: orchestration.Parameters{
				Strategy: orchestration.StrategySpec{
					Type:     orchestration.ParallelStrategy,
					Schedule: orchestration.Immediate,
				},
			},
		})
		require.NoError(t, err)

		svc := kyma.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), &testExecutor{}, resolver, poolingInterval, logrus.New())

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Succeeded, o.State)

	})

	t.Run("DryRun", func(t *testing.T) {
		// given
		store := storage.NewMemoryStorage()

		resolver := &automock.RuntimeResolver{}
		defer resolver.AssertExpectations(t)
		resolver.On("Resolve", orchestration.TargetSpec{}).Return([]orchestration.Runtime{}, nil).Once()

		id := "id"
		err := store.Orchestrations().Insert(internal.Orchestration{
			OrchestrationID: id,
			State:           orchestration.Pending,
			Parameters: orchestration.Parameters{
				DryRun: true,
			}})
		require.NoError(t, err)

		svc := kyma.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), nil, resolver, poolingInterval, logrus.New())

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Succeeded, o.State)

	})

	t.Run("InProgressWithRuntimeOperations", func(t *testing.T) {
		// given
		store := storage.NewMemoryStorage()

		resolver := &automock.RuntimeResolver{}
		defer resolver.AssertExpectations(t)

		id := "id"

		upgradeOperation := internal.UpgradeKymaOperation{
			Operation: internal.Operation{
				ID:                     id,
				Version:                0,
				CreatedAt:              time.Now(),
				UpdatedAt:              time.Now(),
				InstanceID:             "",
				ProvisionerOperationID: "",
				State:                  orchestration.Succeeded,
				Description:            "operation created",
			},
			RuntimeOperation: orchestration.RuntimeOperation{
				Runtime: orchestration.Runtime{
					RuntimeID:    id,
					SubAccountID: "sub",
				},
				DryRun: false,
			},
			ProvisioningParameters: "",
			InputCreator:           nil,
		}
		err := store.Operations().InsertUpgradeKymaOperation(upgradeOperation)
		require.NoError(t, err)

		givenO := internal.Orchestration{
			OrchestrationID: id,
			State:           orchestration.InProgress,
			Parameters: orchestration.Parameters{
				Strategy: orchestration.StrategySpec{
					Type:     orchestration.ParallelStrategy,
					Schedule: orchestration.Immediate,
				}},
		}
		err = store.Orchestrations().Insert(givenO)
		require.NoError(t, err)

		svc := kyma.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), &testExecutor{}, resolver, poolingInterval, logrus.New())

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Succeeded, o.State)
	})

	t.Run("Canceled", func(t *testing.T) {
		// given
		store := storage.NewMemoryStorage()

		resolver := &automock.RuntimeResolver{}
		defer resolver.AssertExpectations(t)

		id := "id"
		err := store.Orchestrations().Insert(internal.Orchestration{
			OrchestrationID: id,
			State:           orchestration.Canceling,
			Parameters: orchestration.Parameters{Strategy: orchestration.StrategySpec{
				Type:     orchestration.ParallelStrategy,
				Schedule: orchestration.Immediate,
			}},
		})

		require.NoError(t, err)
		err = store.Operations().InsertUpgradeKymaOperation(internal.UpgradeKymaOperation{
			Operation: internal.Operation{
				ID:              id,
				OrchestrationID: id,
				State:           orchestration.Pending,
			},
		})

		svc := kyma.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), &testExecutor{}, resolver, poolingInterval, logrus.New())

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Canceled, o.State)

		op, err := store.Operations().GetUpgradeKymaOperationByID(id)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Canceled, string(op.State))
	})
}

type testExecutor struct{}

func (t *testExecutor) Execute(opID string) (time.Duration, error) {
	return 0, nil
}
