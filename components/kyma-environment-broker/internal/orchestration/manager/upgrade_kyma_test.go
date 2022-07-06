package manager_test

import (
	"errors"
	"testing"
	"time"

	internalOrchestration "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration"

	"github.com/stretchr/testify/assert"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/notification"
	notificationAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/notification/mocks"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/manager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
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
		notificationBuilder.On("NewBundle", mock.Anything, notificationParas).Return(bundle, nil).Once()
		bundle.On("CreateNotificationEvent").Return(nil).Once()

		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), nil,
			resolver, 20*time.Millisecond, nil, logrus.New(), k8sClient, &orchestrationConfig, notificationBuilder)

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
					Type:     orchestration.ParallelStrategy,
					Schedule: time.Now().Format(time.RFC3339),
				},
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
		notificationBuilder.On("DisabledCheck").Return(false).Once()
		notificationBuilder.On("NewBundle", id, notificationParas).Return(bundle, nil).Once()
		bundle.On("CreateNotificationEvent").Return(nil).Once()

		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), &testExecutor{},
			resolver, poolingInterval, nil, logrus.New(), k8sClient, &orchestrationConfig, notificationBuilder)

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
				Kyma:       &orchestration.KymaParameters{Version: ""},
				Kubernetes: &orchestration.KubernetesParameters{KubernetesVersion: ""},
			}})
		require.NoError(t, err)

		notificationBuilder := &notificationAutomock.BundleBuilder{}
		notificationBuilder.On("DisabledCheck").Return(false).Once()

		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), nil,
			resolver, poolingInterval, nil, logrus.New(), k8sClient, &orchestrationConfig, notificationBuilder)

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
		err := store.Operations().InsertUpgradeKymaOperation(upgradeOperation)
		require.NoError(t, err)

		givenO := internal.Orchestration{
			OrchestrationID: id,
			State:           orchestration.InProgress,
			Type:            orchestration.UpgradeKymaOrchestration,
			Parameters: orchestration.Parameters{
				Strategy: orchestration.StrategySpec{
					Type:     orchestration.ParallelStrategy,
					Schedule: time.Now().Format(time.RFC3339),
				}},
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
		notificationBuilder.On("DisabledCheck").Return(false).Once()
		notificationBuilder.On("NewBundle", id, notificationParas).Return(bundle, nil).Once()
		bundle.On("CreateNotificationEvent").Return(nil).Once()

		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), &testExecutor{},
			resolver, poolingInterval, nil, logrus.New(), k8sClient, &orchestrationConfig, notificationBuilder)

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
				Schedule: time.Now().Format(time.RFC3339),
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

		notificationParas := notification.NotificationParams{
			OrchestrationID: id,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("DisabledCheck").Return(false)
		notificationBuilder.On("NewBundle", id, notificationParas).Return(bundle, nil).Once()
		bundle.On("CancelNotificationEvent").Return(nil).Once()

		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), &testExecutor{},
			resolver, poolingInterval, nil, logrus.New(), k8sClient, &orchestrationConfig, notificationBuilder)

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

		id := "id"
		opId := "op-" + id
		err := store.Orchestrations().Insert(internal.Orchestration{
			OrchestrationID: id,
			State:           orchestration.Retrying,
			Type:            orchestration.UpgradeKymaOrchestration,
			Parameters: orchestration.Parameters{Strategy: orchestration.StrategySpec{
				Type:     orchestration.ParallelStrategy,
				Schedule: time.Now().Format(time.RFC3339),
				Parallel: orchestration.ParallelStrategySpec{Workers: 2},
			}},
		})
		require.NoError(t, err)

		err = store.Operations().InsertUpgradeKymaOperation(internal.UpgradeKymaOperation{
			Operation: internal.Operation{
				ID:              opId,
				OrchestrationID: id,
				State:           orchestration.Retrying,
			},
			RuntimeOperation: orchestration.RuntimeOperation{
				ID:      opId,
				Runtime: orchestration.Runtime{},
				DryRun:  false,
			},
			InputCreator: nil,
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
		notificationBuilder.On("DisabledCheck").Return(false).Once()
		notificationBuilder.On("NewBundle", id, notificationParas).Return(bundle, nil).Once()
		bundle.On("CreateNotificationEvent").Return(nil).Once()

		executor := retryTestExecutor{
			store:       store,
			upgradeType: orchestration.UpgradeKymaOrchestration,
		}
		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), &executor,
			resolver, poolingInterval, nil, logrus.New(), k8sClient, &orchestrationConfig, notificationBuilder)

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
				Type:     orchestration.ParallelStrategy,
				Schedule: time.Now().Format(time.RFC3339),
				Parallel: orchestration.ParallelStrategySpec{Workers: 2},
			}},
		})
		require.NoError(t, err)

		err = store.Operations().InsertUpgradeKymaOperation(internal.UpgradeKymaOperation{
			Operation: internal.Operation{
				ID:              opId,
				OrchestrationID: id,
				State:           orchestration.Retrying,
			},
			RuntimeOperation: orchestration.RuntimeOperation{
				ID:      opId,
				Runtime: orchestration.Runtime{},
				DryRun:  false,
			},
			InputCreator: nil,
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
		notificationBuilder.On("DisabledCheck").Return(false).Once()
		notificationBuilder.On("NewBundle", id, notificationParas).Return(bundle, nil).Once()
		bundle.On("CreateNotificationEvent").Return(nil).Once()

		executor := retryTestExecutor{
			store:       store,
			upgradeType: orchestration.UpgradeKymaOrchestration,
		}
		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), &executor,
			resolver, poolingInterval, nil, logrus.New(), k8sClient, &orchestrationConfig, notificationBuilder)

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

	return 0, errors.New("unknown upgrade type")
}

func (t *retryTestExecutor) Reschedule(operationID string, maintenanceWindowBegin, maintenanceWindowEnd time.Time) error {
	return nil
}
