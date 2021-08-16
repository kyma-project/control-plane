package manager_test

import (
	"k8s.io/client-go/kubernetes"

	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/manager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const poolingInterval = 20 * time.Millisecond
const kubeconfigRaw = "kubeconfig"

type Client struct {
	Clientset kubernetes.Interface
}

func TestUpgradeKymaManager_Execute(t *testing.T) {
	k8sClient := fake.NewFakeClient()
	configNamespace := "default"
	configName := "policyConfig"

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

		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), nil,
			resolver, 20*time.Millisecond, nil, logrus.New(), k8sClient, configNamespace, configName)

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

		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), &testExecutor{},
			resolver, poolingInterval, nil, logrus.New(), k8sClient, configNamespace, configName)

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

		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), nil,
			resolver, poolingInterval, nil, logrus.New(), k8sClient, configNamespace, configName)

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
			Parameters: orchestration.Parameters{
				Strategy: orchestration.StrategySpec{
					Type:     orchestration.ParallelStrategy,
					Schedule: orchestration.Immediate,
				}},
		}
		err = store.Orchestrations().Insert(givenO)
		require.NoError(t, err)

		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), &testExecutor{},
			resolver, poolingInterval, nil, logrus.New(), k8sClient, configNamespace, configName)

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

		svc := manager.NewUpgradeKymaManager(store.Orchestrations(), store.Operations(), store.Instances(), &testExecutor{},
			resolver, poolingInterval, nil, logrus.New(), k8sClient, configNamespace, configName)

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

func (t *testExecutor) Reschedule(operationID string, maintenanceWindowBegin, maintenanceWindowEnd time.Time) error {
	return nil
}
