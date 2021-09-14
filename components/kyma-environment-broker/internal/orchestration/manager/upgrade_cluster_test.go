package manager_test

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/manager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestUpgradeClusterManager_Execute(t *testing.T) {
	k8sClient := fake.NewFakeClient()
	configNamespace := "default"
	configName := "policyConfig"
	kebConfig := broker.KEBConfig{
		KubernetesVersion: "1.22",
		KymaVersion:       "1.24.5",
	}

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

		svc := manager.NewUpgradeClusterManager(store.Orchestrations(), store.Operations(), store.Instances(), nil,
			resolver, 20*time.Millisecond, logrus.New(), k8sClient, configNamespace, configName, &kebConfig)

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

		svc := manager.NewUpgradeClusterManager(store.Orchestrations(), store.Operations(), store.Instances(), &testExecutor{},
			resolver, poolingInterval, logrus.New(), k8sClient, configNamespace, configName, &kebConfig)

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

		svc := manager.NewUpgradeClusterManager(store.Orchestrations(), store.Operations(), store.Instances(), nil,
			resolver, poolingInterval, logrus.New(), k8sClient, configNamespace, configName, &kebConfig)

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

		upgradeOperation := internal.UpgradeClusterOperation{
			Operation: internal.Operation{
				ID:                     id,
				Version:                0,
				CreatedAt:              time.Now(),
				UpdatedAt:              time.Now(),
				InstanceID:             "",
				ProvisionerOperationID: "",
				OrchestrationID:        id,
				State:                  orchestration.Succeeded,
				Description:            "operation created",
				ProvisioningParameters: internal.ProvisioningParameters{},
			},
			RuntimeOperation: orchestration.RuntimeOperation{
				Runtime: orchestration.Runtime{
					RuntimeID:    id,
					SubAccountID: "sub",
				},
				DryRun: false,
			},
			InputCreator: nil,
		}
		err := store.Operations().InsertUpgradeClusterOperation(upgradeOperation)
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

		svc := manager.NewUpgradeClusterManager(store.Orchestrations(), store.Operations(), store.Instances(), &testExecutor{},
			resolver, poolingInterval, logrus.New(), k8sClient, configNamespace, configName, &kebConfig)

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
		err = store.Operations().InsertUpgradeClusterOperation(internal.UpgradeClusterOperation{
			Operation: internal.Operation{
				ID:              id,
				OrchestrationID: id,
				State:           orchestration.Pending,
			},
		})
		require.NoError(t, err)

		svc := manager.NewUpgradeClusterManager(store.Orchestrations(), store.Operations(), store.Instances(), &testExecutor{}, resolver,
			poolingInterval, logrus.New(), k8sClient, configNamespace, configName, &kebConfig)

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Canceled, o.State)

		op, err := store.Operations().GetUpgradeClusterOperationByID(id)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Canceled, string(op.State))
	})
}
