package manager_test

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/notification"
	notificationAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/notification/mocks"
	internalOrchestration "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/manager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestUpgradeClusterManager_Execute(t *testing.T) {
	k8sClient := fake.NewFakeClient()
	orchestrationConfig := internalOrchestration.Config{
		KymaVersion:       "1.24.5",
		KubernetesVersion: "1.22",
		Namespace:         "default",
		Name:              "policyConfig",
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
		err := store.Orchestrations().Insert(internal.Orchestration{
			OrchestrationID: id,
			State:           orchestration.Pending,
			Parameters: orchestration.Parameters{
				Kyma:       &orchestration.KymaParameters{Version: ""},
				Kubernetes: &orchestration.KubernetesParameters{KubernetesVersion: ""},
			},
		})
		require.NoError(t, err)

		notificationTenants := []notification.NotificationTenant{
			{
				InstanceID: mock.Anything,
				StartDate:  mock.Anything,
				EndDate:    mock.Anything,
			},
		}
		notificationParas := notification.NotificationParams{
			OrchestrationID: id,
			EventType:       mock.Anything,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("DisabledCheck").Return(false).Once()
		notificationBuilder.On("NewBundle", id, notificationParas).Return(bundle, nil).Once()
		bundle.On("CreateNotificationEvent").Return(nil).Once()

		svc := manager.NewUpgradeClusterManager(store.Orchestrations(), store.Operations(), store.Instances(), nil,
			resolver, 20*time.Millisecond, logrus.New(), k8sClient, orchestrationConfig, notificationBuilder, 1000)

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
			Type:            orchestration.UpgradeClusterOrchestration,
			Parameters: orchestration.Parameters{
				Strategy: orchestration.StrategySpec{
					Type:     orchestration.ParallelStrategy,
					Schedule: orchestration.Immediate,
				},
			},
		})
		require.NoError(t, err)

		notificationTenants := []notification.NotificationTenant{}
		notificationParas := notification.NotificationParams{
			OrchestrationID: id,
			EventType:       notification.KubernetesMaintenanceNumber,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("DisabledCheck").Return(false).Once()
		notificationBuilder.On("NewBundle", id, notificationParas).Return(bundle, nil).Once()
		bundle.On("CreateNotificationEvent").Return(nil).Once()

		svc := manager.NewUpgradeClusterManager(store.Orchestrations(), store.Operations(), store.Instances(), &testExecutor{},
			resolver, poolingInterval, logrus.New(), k8sClient, orchestrationConfig, notificationBuilder, 1000)

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
				DryRun:     true,
				Kubernetes: &orchestration.KubernetesParameters{KubernetesVersion: ""},
				Kyma:       &orchestration.KymaParameters{Version: ""},
			}})
		require.NoError(t, err)

		notificationBuilder := &notificationAutomock.BundleBuilder{}
		notificationBuilder.On("DisabledCheck").Return(false).Once()

		svc := manager.NewUpgradeClusterManager(store.Orchestrations(), store.Operations(), store.Instances(), nil,
			resolver, poolingInterval, logrus.New(), k8sClient, orchestrationConfig, notificationBuilder, 1000)

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
				RuntimeOperation: orchestration.RuntimeOperation{
					Runtime: orchestration.Runtime{
						RuntimeID:    id,
						SubAccountID: "sub",
					},
					DryRun: false,
				},
				InputCreator: nil,
			},
		}
		err := store.Operations().InsertUpgradeClusterOperation(upgradeOperation)
		require.NoError(t, err)

		givenO := internal.Orchestration{
			OrchestrationID: id,
			State:           orchestration.InProgress,
			Type:            orchestration.UpgradeClusterOrchestration,
			Parameters: orchestration.Parameters{
				Strategy: orchestration.StrategySpec{
					Type:     orchestration.ParallelStrategy,
					Schedule: orchestration.Immediate,
				}},
		}
		err = store.Orchestrations().Insert(givenO)
		require.NoError(t, err)

		notificationTenants := []notification.NotificationTenant{}
		notificationParas := notification.NotificationParams{
			OrchestrationID: id,
			EventType:       notification.KubernetesMaintenanceNumber,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("DisabledCheck").Return(false).Once()
		notificationBuilder.On("NewBundle", id, notificationParas).Return(bundle, nil).Once()
		bundle.On("CreateNotificationEvent").Return(nil).Once()

		svc := manager.NewUpgradeClusterManager(store.Orchestrations(), store.Operations(), store.Instances(), &testExecutor{},
			resolver, poolingInterval, logrus.New(), k8sClient, orchestrationConfig, notificationBuilder, 1000)

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

		notificationParas := notification.NotificationParams{
			OrchestrationID: id,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("DisabledCheck").Return(false)
		notificationBuilder.On("NewBundle", id, notificationParas).Return(bundle, nil).Once()
		bundle.On("CancelNotificationEvent").Return(nil).Once()

		svc := manager.NewUpgradeClusterManager(store.Orchestrations(), store.Operations(), store.Instances(), &testExecutor{}, resolver,
			poolingInterval, logrus.New(), k8sClient, orchestrationConfig, notificationBuilder, 1000)

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

	t.Run("Retrying failed orchestration", func(t *testing.T) {
		// given
		store := storage.NewMemoryStorage()

		resolver := &automock.RuntimeResolver{}
		defer resolver.AssertExpectations(t)

		id := "id"
		opId := "op-" + id
		err := store.Orchestrations().Insert(internal.Orchestration{
			OrchestrationID: id,
			State:           orchestration.Retrying,
			Type:            orchestration.UpgradeClusterOrchestration,
			Parameters: orchestration.Parameters{Strategy: orchestration.StrategySpec{
				Type:     orchestration.ParallelStrategy,
				Schedule: orchestration.Immediate,
				Parallel: orchestration.ParallelStrategySpec{Workers: 2},
			}},
		})
		require.NoError(t, err)

		err = store.Operations().InsertUpgradeClusterOperation(internal.UpgradeClusterOperation{
			Operation: internal.Operation{
				ID:              opId,
				OrchestrationID: id,
				State:           orchestration.Retrying,
				RuntimeOperation: orchestration.RuntimeOperation{
					ID:      opId,
					Runtime: orchestration.Runtime{},
					DryRun:  false,
				},
				InputCreator: nil,
			},
		})
		require.NoError(t, err)

		notificationTenants := []notification.NotificationTenant{
			{
				StartDate: time.Now().Format("2006-01-02 15:04:05"),
			},
		}
		notificationParas := notification.NotificationParams{
			OrchestrationID: id,
			EventType:       notification.KubernetesMaintenanceNumber,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("DisabledCheck").Return(false).Once()
		notificationBuilder.On("NewBundle", id, notificationParas).Return(bundle, nil).Once()
		bundle.On("CreateNotificationEvent").Return(nil).Once()

		executor := retryTestExecutor{
			store:       store,
			upgradeType: orchestration.UpgradeClusterOrchestration,
		}
		svc := manager.NewUpgradeClusterManager(store.Orchestrations(), store.Operations(), store.Instances(), &executor, resolver,
			poolingInterval, logrus.New(), k8sClient, orchestrationConfig, notificationBuilder, 1000)

		_, err = store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		_, err = store.Operations().GetUpgradeClusterOperationByID("op-id")
		require.NoError(t, err)

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Failed, o.State)

		op, err := store.Operations().GetUpgradeClusterOperationByID(opId)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Retrying, string(op.State))
	})

	t.Run("Retrying failed orchestration and create a new operation on same instanceID", func(t *testing.T) {
		// given
		store := storage.NewMemoryStorage()

		resolver := &automock.RuntimeResolver{}
		defer resolver.AssertExpectations(t)

		id := "id"
		opId := "op-" + id
		instanceID := opId + "-1234"
		runtimeID := opId + "-5678"

		resolver.On("Resolve", orchestration.TargetSpec{
			Include: []orchestration.RuntimeTarget{
				{RuntimeID: opId},
			},
			Exclude: nil,
		}).Return([]orchestration.Runtime{{
			InstanceID: instanceID,
			RuntimeID:  runtimeID,
		}}, nil)

		err := store.Instances().Insert(internal.Instance{
			InstanceID: instanceID,
			RuntimeID:  runtimeID,
		})
		require.NoError(t, err)

		err = store.Orchestrations().Insert(internal.Orchestration{
			OrchestrationID: id,
			State:           orchestration.Retrying,
			Type:            orchestration.UpgradeClusterOrchestration,
			Parameters: orchestration.Parameters{Strategy: orchestration.StrategySpec{
				Type:     orchestration.ParallelStrategy,
				Schedule: orchestration.Immediate,
				Parallel: orchestration.ParallelStrategySpec{Workers: 2},
			},
				Targets: orchestration.TargetSpec{
					Include: []orchestration.RuntimeTarget{
						{RuntimeID: opId},
					},
					Exclude: nil,
				},
				RetryOperations: []string{"op-id"}},
		})

		require.NoError(t, err)

		err = store.Operations().InsertUpgradeClusterOperation(internal.UpgradeClusterOperation{
			Operation: internal.Operation{
				ID:              opId,
				OrchestrationID: id,
				State:           orchestration.Failed,
				InstanceID:      instanceID,
				Type:            internal.OperationTypeUpgradeCluster,
				RuntimeOperation: orchestration.RuntimeOperation{
					ID: opId,
					Runtime: orchestration.Runtime{
						InstanceID: instanceID,
						RuntimeID:  runtimeID,
					},
					DryRun: false,
				},
			},
		})
		require.NoError(t, err)

		notificationBuilder := &notificationAutomock.BundleBuilder{}
		notificationBuilder.On("DisabledCheck").Return(true).Once()

		executor := retryTestExecutor{
			store:       store,
			upgradeType: orchestration.UpgradeClusterOrchestration,
		}
		svc := manager.NewUpgradeClusterManager(store.Orchestrations(), store.Operations(), store.Instances(), &executor, resolver,
			poolingInterval, logrus.New(), k8sClient, orchestrationConfig, notificationBuilder, 1000)

		_, err = store.Operations().GetUpgradeClusterOperationByID(opId)
		require.NoError(t, err)

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Succeeded, o.State)

		op, err := store.Operations().GetUpgradeClusterOperationByID(opId)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Failed, string(op.State))

		//verify a new operation with same instanceID is created
		ops, _, _, err := store.Operations().ListUpgradeClusterOperationsByOrchestrationID(id, dbmodel.OperationFilter{})
		require.NoError(t, err)
		assert.Equal(t, 2, len(ops))

		for _, op := range ops {
			if op.Operation.ID != opId {
				assert.Equal(t, orchestration.Succeeded, string(op.State))
				assert.Equal(t, instanceID, string(op.Operation.InstanceID))
				assert.Equal(t, internal.OperationTypeUpgradeCluster, op.Operation.Type)
			}
		}
	})

	t.Run("Retrying resumed in progress orchestration", func(t *testing.T) {
		// given
		store := storage.NewMemoryStorage()

		resolver := &automock.RuntimeResolver{}
		defer resolver.AssertExpectations(t)

		id := "id"
		opId := "op-" + id
		err := store.Orchestrations().Insert(internal.Orchestration{
			OrchestrationID: id,
			State:           orchestration.InProgress,
			Type:            orchestration.UpgradeClusterOrchestration,
			Parameters: orchestration.Parameters{Strategy: orchestration.StrategySpec{
				Type:     orchestration.ParallelStrategy,
				Schedule: orchestration.Immediate,
				Parallel: orchestration.ParallelStrategySpec{Workers: 2},
			}},
		})
		require.NoError(t, err)

		err = store.Operations().InsertUpgradeClusterOperation(internal.UpgradeClusterOperation{
			Operation: internal.Operation{
				ID:              opId,
				OrchestrationID: id,
				State:           orchestration.Retrying,
				RuntimeOperation: orchestration.RuntimeOperation{
					ID:      opId,
					Runtime: orchestration.Runtime{},
					DryRun:  false,
				},
				InputCreator: nil,
			},
		})
		require.NoError(t, err)

		notificationTenants := []notification.NotificationTenant{
			{
				StartDate: time.Now().Format("2006-01-02 15:04:05"),
			},
		}
		notificationParas := notification.NotificationParams{
			OrchestrationID: id,
			EventType:       notification.KubernetesMaintenanceNumber,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("DisabledCheck").Return(false).Once()
		notificationBuilder.On("NewBundle", id, notificationParas).Return(bundle, nil).Once()
		bundle.On("CreateNotificationEvent").Return(nil).Once()

		executor := retryTestExecutor{
			store:       store,
			upgradeType: orchestration.UpgradeClusterOrchestration,
		}
		svc := manager.NewUpgradeClusterManager(store.Orchestrations(), store.Operations(), store.Instances(), &executor, resolver,
			poolingInterval, logrus.New(), k8sClient, orchestrationConfig, notificationBuilder, 1000)

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Succeeded, o.State)

		op, err := store.Operations().GetUpgradeClusterOperationByID(opId)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Succeeded, string(op.State))
	})
}
