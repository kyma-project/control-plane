package manager_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/notification"
	internalOrchestration "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	notificationAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/notification/mocks"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/manager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const (
	poolingInterval    = 20 * time.Millisecond
	defaultKymaVersion = "1.24.5"
)

func TestUpgradeKymaManager_Execute(t *testing.T) {
	k8sClient := fake.NewFakeClient()
	orchestrationConfig := internalOrchestration.Config{
		KymaVersion:       defaultKymaVersion,
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
			Type:            orchestration.UpgradeKymaOrchestration,
			Parameters: orchestration.Parameters{
				Kyma:       &orchestration.KymaParameters{Version: ""},
				Kubernetes: &orchestration.KubernetesParameters{KubernetesVersion: ""},
				Strategy: orchestration.StrategySpec{
					ScheduleTime: time.Time{},
				},
				Notification: true,
			},
		})
		require.NoError(t, err)

		notificationTenants := []notification.NotificationTenant{}
		notificationParas := notification.NotificationParams{
			OrchestrationID: id,
			EventType:       notification.KymaMaintenanceNumber,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("NewBundle", mock.Anything, notificationParas).Return(bundle, nil).Once()
		bundle.On("CreateNotificationEvent").Return(nil).Once()

		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), nil,
			resolver, 20*time.Millisecond, logrus.New(), k8sClient, &orchestrationConfig, notificationBuilder, 1000)

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
			Type:            orchestration.UpgradeKymaOrchestration,
			Parameters: orchestration.Parameters{
				Strategy: orchestration.StrategySpec{
					Type:         orchestration.ParallelStrategy,
					Schedule:     time.Now().Format(time.RFC3339),
					ScheduleTime: time.Time{},
				},
				Notification: true,
			},
		})
		require.NoError(t, err)

		notificationTenants := []notification.NotificationTenant{}
		notificationParas := notification.NotificationParams{
			OrchestrationID: id,
			EventType:       notification.KymaMaintenanceNumber,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("NewBundle", id, notificationParas).Return(bundle, nil).Once()
		bundle.On("CreateNotificationEvent").Return(nil).Once()

		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), &testExecutor{},
			resolver, poolingInterval, logrus.New(), k8sClient, &orchestrationConfig, notificationBuilder, 1000)

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
			Type:            orchestration.UpgradeKymaOrchestration,
			Parameters: orchestration.Parameters{
				DryRun:     true,
				Kyma:       &orchestration.KymaParameters{Version: ""},
				Kubernetes: &orchestration.KubernetesParameters{KubernetesVersion: ""},
				Strategy: orchestration.StrategySpec{
					ScheduleTime: time.Time{},
				},
				Notification: true,
			}})
		require.NoError(t, err)

		notificationTenants := []notification.NotificationTenant{}
		notificationParas := notification.NotificationParams{
			OrchestrationID: id,
			EventType:       notification.KymaMaintenanceNumber,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("NewBundle", id, notificationParas).Return(bundle, nil).Once()
		bundle.On("CreateNotificationEvent").Return(nil).Once()

		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), nil,
			resolver, poolingInterval, logrus.New(), k8sClient, &orchestrationConfig, notificationBuilder, 1000)

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
		err := store.Operations().InsertUpgradeKymaOperation(upgradeOperation)
		require.NoError(t, err)

		givenO := internal.Orchestration{
			OrchestrationID: id,
			State:           orchestration.InProgress,
			Type:            orchestration.UpgradeKymaOrchestration,
			Parameters: orchestration.Parameters{
				Strategy: orchestration.StrategySpec{
					Type:         orchestration.ParallelStrategy,
					Schedule:     time.Now().Format(time.RFC3339),
					ScheduleTime: time.Time{},
				},
				Notification: true,
			},
		}
		err = store.Orchestrations().Insert(givenO)
		require.NoError(t, err)

		notificationTenants := []notification.NotificationTenant{}
		notificationParas := notification.NotificationParams{
			OrchestrationID: id,
			EventType:       notification.KymaMaintenanceNumber,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("NewBundle", id, notificationParas).Return(bundle, nil).Once()
		bundle.On("CreateNotificationEvent").Return(nil).Once()

		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), &testExecutor{},
			resolver, poolingInterval, logrus.New(), k8sClient, &orchestrationConfig, notificationBuilder, 1000)

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
				Type:         orchestration.ParallelStrategy,
				Schedule:     time.Now().Format(time.RFC3339),
				ScheduleTime: time.Time{},
			},
				Notification: true,
			},
		})

		require.NoError(t, err)
		err = store.Operations().InsertUpgradeKymaOperation(internal.UpgradeKymaOperation{
			Operation: internal.Operation{
				ID:              id,
				OrchestrationID: id,
				State:           orchestration.Pending,
				RuntimeOperation: orchestration.RuntimeOperation{
					Runtime: orchestration.Runtime{
						RuntimeID:    id,
						SubAccountID: "sub",
					},
					DryRun: false,
				},
			},
		})

		notificationParas := notification.NotificationParams{
			OrchestrationID: id,
			Tenants:         []notification.NotificationTenant{},
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("NewBundle", id, notificationParas).Return(bundle, nil).Once()
		bundle.On("CancelNotificationEvent").Return(nil).Once()

		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), &testExecutor{},
			resolver, poolingInterval, logrus.New(), k8sClient, &orchestrationConfig, notificationBuilder, 1000)

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

	t.Run("Retrying failed orchestration", func(t *testing.T) {
		// given
		store := storage.NewMemoryStorage()

		resolver := &automock.RuntimeResolver{}
		defer resolver.AssertExpectations(t)

		resolver.On("Resolve", orchestration.TargetSpec{
			Include: nil,
			Exclude: nil,
		}).Return([]orchestration.Runtime{}, nil)

		id := "id"
		opId := "op-" + id
		err := store.Orchestrations().Insert(internal.Orchestration{
			OrchestrationID: id,
			State:           orchestration.Retrying,
			Type:            orchestration.UpgradeKymaOrchestration,
			Parameters: orchestration.Parameters{
				Strategy: orchestration.StrategySpec{
					Type:         orchestration.ParallelStrategy,
					Schedule:     time.Now().Format(time.RFC3339),
					Parallel:     orchestration.ParallelStrategySpec{Workers: 2},
					ScheduleTime: time.Time{},
				},
				RetryOperation: orchestration.RetryOperationParameters{
					RetryOperations: []string{"op-id"},
				},
				Notification: true,
			},
		})
		require.NoError(t, err)

		err = store.Operations().InsertUpgradeKymaOperation(internal.UpgradeKymaOperation{
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
			EventType:       notification.KymaMaintenanceNumber,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("NewBundle", id, notificationParas).Return(bundle, nil).Once()
		bundle.On("CreateNotificationEvent").Return(nil).Once()

		executor := retryTestExecutor{
			store:       store,
			upgradeType: orchestration.UpgradeKymaOrchestration,
		}
		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), &executor,
			resolver, poolingInterval, logrus.New(), k8sClient, &orchestrationConfig, notificationBuilder, 1000)

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Succeeded, o.State)

		op, err := store.Operations().GetUpgradeKymaOperationByID(opId)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Retrying, string(op.State))
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
			Type:            orchestration.UpgradeKymaOrchestration,
			Parameters: orchestration.Parameters{Strategy: orchestration.StrategySpec{
				Type:         orchestration.ParallelStrategy,
				Schedule:     time.Now().Format(time.RFC3339),
				Parallel:     orchestration.ParallelStrategySpec{Workers: 2},
				ScheduleTime: time.Time{},
			},
				Notification: true,
			},
		})
		require.NoError(t, err)

		err = store.Operations().InsertUpgradeKymaOperation(internal.UpgradeKymaOperation{
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
				Type:         internal.OperationTypeUpgradeKyma,
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
			EventType:       notification.KymaMaintenanceNumber,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("NewBundle", id, notificationParas).Return(bundle, nil).Once()
		bundle.On("CreateNotificationEvent").Return(nil).Once()

		executor := retryTestExecutor{
			store:       store,
			upgradeType: orchestration.UpgradeKymaOrchestration,
		}
		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), &executor,
			resolver, poolingInterval, logrus.New(), k8sClient, &orchestrationConfig, notificationBuilder, 1000)

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Succeeded, o.State)

		op, err := store.Operations().GetUpgradeKymaOperationByID(opId)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Succeeded, string(op.State))
	})

	t.Run("Retrying failed orchestration with --schedule immediate and create a new operation on same instanceID", func(t *testing.T) {
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
		err = store.Orchestrations().Insert(
			internal.Orchestration{
				OrchestrationID: id,
				State:           orchestration.Retrying,
				Type:            orchestration.UpgradeClusterOrchestration,
				Parameters: orchestration.Parameters{
					Strategy: orchestration.StrategySpec{
						Type:         orchestration.ParallelStrategy,
						Schedule:     time.Now().Format(time.RFC3339),
						Parallel:     orchestration.ParallelStrategySpec{Workers: 2},
						ScheduleTime: time.Time{},
					},
					Kyma: &orchestration.KymaParameters{Version: ""},
					Targets: orchestration.TargetSpec{
						Include: []orchestration.RuntimeTarget{
							{RuntimeID: opId},
						},
						Exclude: nil,
					},
					RetryOperation: orchestration.RetryOperationParameters{
						RetryOperations: []string{"op-id"},
						Immediate:       true,
					},
					Notification: false,
				},
			})
		require.NoError(t, err)

		err = store.Operations().InsertUpgradeKymaOperation(internal.UpgradeKymaOperation{
			Operation: internal.Operation{
				ID:              opId,
				OrchestrationID: id,
				State:           orchestration.Failed,
				InstanceID:      instanceID,
				Type:            internal.OperationTypeUpgradeKyma,
				RuntimeOperation: orchestration.RuntimeOperation{
					ID: opId,
					Runtime: orchestration.Runtime{
						InstanceID: instanceID,
						RuntimeID:  runtimeID},
					DryRun: false,
				},
			},
		})
		require.NoError(t, err)

		notificationBuilder := &notificationAutomock.BundleBuilder{}

		executor := retryTestExecutor{
			store:       store,
			upgradeType: orchestration.UpgradeKymaOrchestration,
		}
		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), &executor,
			resolver, poolingInterval, logrus.New(), k8sClient, &orchestrationConfig, notificationBuilder, 1000)

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Succeeded, o.State)

		op, err := store.Operations().GetUpgradeKymaOperationByID(opId)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Failed, string(op.State))

		//verify a new operation with same instanceID is created
		ops, _, _, err := store.Operations().ListUpgradeKymaOperationsByOrchestrationID(id, dbmodel.OperationFilter{})
		require.NoError(t, err)
		assert.Equal(t, 2, len(ops))

		for _, op := range ops {
			if op.Operation.ID != opId {
				assert.Equal(t, op.Operation.RuntimeOperation.MaintenanceWindowBegin, time.Time{})
				assert.Equal(t, op.Operation.RuntimeOperation.MaintenanceWindowEnd, time.Time{})

				assert.Equal(t, orchestration.Succeeded, string(op.State))
				assert.Equal(t, instanceID, string(op.Operation.InstanceID))
				assert.Equal(t, internal.OperationTypeUpgradeKyma, op.Operation.Type)
			}
		}
	})

	t.Run("Retrying --now failed orchestration with `--schedule maintancewindow`  and create a new operation on same instanceID", func(t *testing.T) {
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
		err = store.Orchestrations().Insert(
			internal.Orchestration{
				OrchestrationID: id,
				State:           orchestration.Retrying,
				Type:            orchestration.UpgradeClusterOrchestration,
				Parameters: orchestration.Parameters{
					Strategy: orchestration.StrategySpec{
						Type:         orchestration.ParallelStrategy,
						Schedule:     time.Now().Format(time.RFC3339),
						Parallel:     orchestration.ParallelStrategySpec{Workers: 2},
						ScheduleTime: time.Time{},
					},
					Kyma: &orchestration.KymaParameters{Version: ""},
					Targets: orchestration.TargetSpec{
						Include: []orchestration.RuntimeTarget{
							{RuntimeID: opId},
						},
						Exclude: nil,
					},
					RetryOperation: orchestration.RetryOperationParameters{
						RetryOperations: []string{"op-id"},
						Immediate:       true,
					},
					Notification: false,
				},
			})
		require.NoError(t, err)

		err = store.Operations().InsertUpgradeKymaOperation(internal.UpgradeKymaOperation{
			Operation: internal.Operation{
				ID:              opId,
				OrchestrationID: id,
				State:           orchestration.Failed,
				InstanceID:      instanceID,
				Type:            internal.OperationTypeUpgradeKyma,
				RuntimeOperation: orchestration.RuntimeOperation{
					ID: opId,
					Runtime: orchestration.Runtime{
						InstanceID: instanceID,
						RuntimeID:  runtimeID},
					DryRun: false,
				},
			},
		})
		require.NoError(t, err)

		notificationBuilder := &notificationAutomock.BundleBuilder{}

		executor := retryTestExecutor{
			store:       store,
			upgradeType: orchestration.UpgradeKymaOrchestration,
		}
		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), &executor,
			resolver, poolingInterval, logrus.New(), k8sClient, &orchestrationConfig, notificationBuilder, 1000)

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Succeeded, o.State)

		op, err := store.Operations().GetUpgradeKymaOperationByID(opId)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Failed, string(op.State))

		//verify a new operation with same instanceID is created
		ops, _, _, err := store.Operations().ListUpgradeKymaOperationsByOrchestrationID(id, dbmodel.OperationFilter{})
		require.NoError(t, err)
		assert.Equal(t, 2, len(ops))

		for _, op := range ops {
			if op.Operation.ID != opId {
				assert.Equal(t, op.Operation.RuntimeOperation.MaintenanceWindowBegin, time.Time{})
				assert.Equal(t, op.Operation.RuntimeOperation.MaintenanceWindowEnd, time.Time{})
				assert.Equal(t, orchestration.Succeeded, string(op.State))
				assert.Equal(t, instanceID, string(op.Operation.InstanceID))
				assert.Equal(t, internal.OperationTypeUpgradeKyma, op.Operation.Type)
			}
		}
	})

	t.Run("Retrying failed orchestration with `--schedule maintancewindow` and without `--now`  and create a new operation on same instanceID", func(t *testing.T) {
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
		err = store.Orchestrations().Insert(
			internal.Orchestration{
				OrchestrationID: id,
				State:           orchestration.Retrying,
				Type:            orchestration.UpgradeClusterOrchestration,
				Parameters: orchestration.Parameters{
					Strategy: orchestration.StrategySpec{
						Type:         orchestration.ParallelStrategy,
						Schedule:     time.Now().Format(time.RFC3339),
						Parallel:     orchestration.ParallelStrategySpec{Workers: 2},
						ScheduleTime: time.Time{},
					},
					Kyma: &orchestration.KymaParameters{Version: ""},
					Targets: orchestration.TargetSpec{
						Include: []orchestration.RuntimeTarget{
							{RuntimeID: opId},
						},
						Exclude: nil,
					},
					RetryOperation: orchestration.RetryOperationParameters{
						RetryOperations: []string{"op-id"},
						Immediate:       false,
					},
					Notification: false,
				},
			})
		require.NoError(t, err)

		err = store.Operations().InsertUpgradeKymaOperation(internal.UpgradeKymaOperation{
			Operation: internal.Operation{
				ID:              opId,
				OrchestrationID: id,
				State:           orchestration.Failed,
				InstanceID:      instanceID,
				Type:            internal.OperationTypeUpgradeKyma,
				RuntimeOperation: orchestration.RuntimeOperation{
					ID: opId,
					Runtime: orchestration.Runtime{
						InstanceID: instanceID,
						RuntimeID:  runtimeID},
					DryRun: false,
				},
			},
		})
		require.NoError(t, err)

		notificationBuilder := &notificationAutomock.BundleBuilder{}

		executor := retryTestExecutor{
			store:       store,
			upgradeType: orchestration.UpgradeKymaOrchestration,
		}
		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), &executor,
			resolver, poolingInterval, logrus.New(), k8sClient, &orchestrationConfig, notificationBuilder, 1000)

		// when
		_, err = svc.Execute(id)
		require.NoError(t, err)

		o, err := store.Orchestrations().GetByID(id)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Succeeded, o.State)

		op, err := store.Operations().GetUpgradeKymaOperationByID(opId)
		require.NoError(t, err)

		assert.Equal(t, orchestration.Failed, string(op.State))

		//verify a new operation with same instanceID is created
		ops, _, _, err := store.Operations().ListUpgradeKymaOperationsByOrchestrationID(id, dbmodel.OperationFilter{})
		require.NoError(t, err)
		assert.Equal(t, 2, len(ops))

		for _, op := range ops {
			if op.Operation.ID != opId {
				assert.Equal(t, op.Operation.RuntimeOperation.MaintenanceWindowBegin, time.Time{})
				assert.Equal(t, op.Operation.RuntimeOperation.MaintenanceWindowEnd, time.Time{})
				assert.Equal(t, orchestration.Succeeded, string(op.State))
				assert.Equal(t, instanceID, string(op.Operation.InstanceID))
				assert.Equal(t, internal.OperationTypeUpgradeKyma, op.Operation.Type)
			}
		}
	})
}

type testExecutor struct{}

func (t *testExecutor) Execute(opID string) (time.Duration, error) {
	return 0, nil
}

func (t *testExecutor) Reschedule(operationID string, maintenanceWindowBegin, maintenanceWindowEnd time.Time) error {
	return nil
}

type retryTestExecutor struct {
	store       storage.BrokerStorage
	upgradeType orchestration.Type
}

func (t *retryTestExecutor) Execute(opID string) (time.Duration, error) {
	switch t.upgradeType {
	case orchestration.UpgradeKymaOrchestration:
		op, err := t.store.Operations().GetUpgradeKymaOperationByID(opID)
		if err != nil {
			return 0, err
		}
		op.State = orchestration.Succeeded
		_, err = t.store.Operations().UpdateUpgradeKymaOperation(*op)

		return 0, err
	case orchestration.UpgradeClusterOrchestration:
		op, err := t.store.Operations().GetUpgradeClusterOperationByID(opID)
		if err != nil {
			return 0, err
		}
		op.State = orchestration.Succeeded
		_, err = t.store.Operations().UpdateUpgradeClusterOperation(*op)

		return 0, err
	}

	return 0, fmt.Errorf("unknown upgrade type")
}

func (t *retryTestExecutor) Reschedule(operationID string, maintenanceWindowBegin, maintenanceWindowEnd time.Time) error {
	return nil
}
