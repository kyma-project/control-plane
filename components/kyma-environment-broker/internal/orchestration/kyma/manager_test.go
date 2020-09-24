package kyma_test

import (
	"testing"
	"time"

	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/automock"
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

		resolver.On("Resolve", internal.TargetSpec{
			Include: nil,
			Exclude: nil,
		}).Return([]internal.Runtime{}, nil)

		id := "id"
		err := store.Orchestrations().Insert(internal.Orchestration{OrchestrationID: id, State: internal.Pending})
		require.NoError(t, err)

		svc := kyma.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), nil, resolver, 20*time.Millisecond, logrus.New())

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, internal.Succeeded, o.State)
	})
	t.Run("InProgress", func(t *testing.T) {
		// given
		store := storage.NewMemoryStorage()

		resolver := &automock.RuntimeResolver{}
		defer resolver.AssertExpectations(t)

		id := "id"
		err := store.Orchestrations().Insert(internal.Orchestration{OrchestrationID: id, State: internal.InProgress})
		require.NoError(t, err)

		svc := kyma.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), nil, resolver, poolingInterval, logrus.New())

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, internal.Succeeded, o.State)

	})

	t.Run("DryRun", func(t *testing.T) {
		// given
		store := storage.NewMemoryStorage()

		resolver := &automock.RuntimeResolver{}
		defer resolver.AssertExpectations(t)
		resolver.On("Resolve", internal.TargetSpec{}).Return([]internal.Runtime{}, nil).Once()

		id := "id"
		err := store.Orchestrations().Insert(internal.Orchestration{
			OrchestrationID: id,
			State: internal.Pending,
			Parameters: internal.OrchestrationParameters{
				DryRun: true,
			}})
		require.NoError(t, err)

		svc := kyma.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), nil, resolver, poolingInterval, logrus.New())

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, internal.Succeeded, o.State)

	})

	t.Run("InProgressWithRuntimeOperations", func(t *testing.T) {
		// given
		store := storage.NewMemoryStorage()

		resolver := &automock.RuntimeResolver{}
		defer resolver.AssertExpectations(t)

		id := "id"

		upgradeOperation := internal.UpgradeKymaOperation{
			RuntimeOperation: internal.RuntimeOperation{
				Operation: internal.Operation{
					ID:                     id,
					Version:                0,
					CreatedAt:              time.Now(),
					UpdatedAt:              time.Now(),
					InstanceID:             "",
					ProvisionerOperationID: "",
					State:                  domain.Succeeded,
					Description:            "operation created",
				},
				RuntimeID:    id,
				SubAccountID: "sub",
				DryRun:       false,
			},
			ProvisioningParameters: "",
			InputCreator:           nil,
		}
		err := store.Operations().InsertUpgradeKymaOperation(upgradeOperation)
		require.NoError(t, err)

		givenO := internal.Orchestration{
			OrchestrationID: id,
			State:           internal.InProgress,
			}
		err = store.Orchestrations().Insert(givenO)
		require.NoError(t, err)

		svc := kyma.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), &testExecutor{}, resolver, poolingInterval, logrus.New())

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, internal.Succeeded, o.State)

	})
}

type testExecutor struct{}

func (t *testExecutor) Execute(opID string) (time.Duration, error) {
	return 0, nil
}
