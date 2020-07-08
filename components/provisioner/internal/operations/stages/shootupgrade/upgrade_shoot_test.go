package shootupgrade

import (
	"errors"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	gardener_mocks "github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/deprovisioning/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestWaitForClusterUpgrade(t *testing.T) {

	clusterName := "shootName"
	runtimeID := "runtimeID"
	tenant := "tenant"

	cluster := model.Cluster{
		ID:     runtimeID,
		Tenant: tenant,
		ClusterConfig: model.GardenerConfig{
			Name: clusterName,
		},
	}

	for _, testCase := range []struct {
		description   string
		mockFunc      func(gardenerClient *gardener_mocks.GardenerClient)
		expectedStage model.OperationStage
		expectedDelay time.Duration
	}{
		{
			description: "should continue waiting if cluster is in processing state",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", clusterName, mock.Anything).Return(fixShootInProcessingState(clusterName), nil)
			},
			expectedStage: model.WaitingForShootUpgrade,
			expectedDelay: 20 * time.Second,
		},
		{
			description: "should continue waiting if cluster is in pending state",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", clusterName, mock.Anything).Return(fixShootInPendingState(clusterName), nil)
			},
			expectedStage: model.WaitingForShootUpgrade,
			expectedDelay: 20 * time.Second,
		},
		{
			description: "should continue waiting if cluster is in error state - the operation will be retried on Gardener side",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", clusterName, mock.Anything).Return(fixShootInErrorState(clusterName), nil)
			},
			expectedStage: model.WaitingForShootUpgrade,
			expectedDelay: 20 * time.Second,
		},
		{
			description: "should continue waiting if last operation not set",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", clusterName, mock.Anything).Return(fixShootInUnknownState(clusterName), nil)
			},
			expectedStage: model.WaitingForShootUpgrade,
			expectedDelay: 20 * time.Second,
		},
		{
			description: "should return finished stage if cluster upgrade has succeeded",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {

				gardenerClient.On("Get", clusterName, mock.Anything).Return(fixShootInSucceededState(clusterName), nil)
			},
			expectedStage: model.FinishedStage,
			expectedDelay: 0,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardener_mocks.GardenerClient{}

			testCase.mockFunc(gardenerClient)

			waitForrShootClusterUpgradeStep := NewWaitForShootClusterUpgradeStep(gardenerClient, model.FinishedStage, time.Minute)
			// when
			result, err := waitForrShootClusterUpgradeStep.Run(cluster, model.Operation{}, logrus.New())

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedStage, result.Stage)
			assert.Equal(t, testCase.expectedDelay, result.Delay)
			gardenerClient.AssertExpectations(t)
		})
	}

	for _, testCase := range []struct {
		description        string
		mockFunc           func(gardenerClient *gardener_mocks.GardenerClient)
		unrecoverableError bool
		cluster            model.Cluster
	}{
		{
			description: "should return error if failed to read Shoot",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", clusterName, mock.Anything).Return(nil, errors.New("some error"))
			},
			unrecoverableError: false,
			cluster:            cluster,
		},
		{
			description: "should return unrecoverable error if Shoot is in failed state",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", clusterName, mock.Anything).Return(fixShootInFailedState(clusterName), nil)
			},
			unrecoverableError: true,
			cluster:            cluster,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardener_mocks.GardenerClient{}

			testCase.mockFunc(gardenerClient)

			waitForClusterCreationStep := NewWaitForShootClusterUpgradeStep(gardenerClient, model.FinishedStage, time.Minute)

			// when
			_, err := waitForClusterCreationStep.Run(testCase.cluster, model.Operation{}, logrus.New())

			// then
			require.Error(t, err)
			nonRecoverable := operations.NonRecoverableError{}
			require.Equal(t, testCase.unrecoverableError, errors.As(err, &nonRecoverable))
			gardenerClient.AssertExpectations(t)
		})
	}
}

func fixShootInProcessingState(name string) *gardener_types.Shoot {
	return fixShoot(name, &gardener_types.LastOperation{
		State: gardencorev1beta1.LastOperationStateProcessing,
	})
}

func fixShootInPendingState(name string) *gardener_types.Shoot {
	return fixShoot(name, &gardener_types.LastOperation{
		State: gardencorev1beta1.LastOperationStatePending,
	})
}

func fixShootInErrorState(name string) *gardener_types.Shoot {
	return fixShoot(name, &gardener_types.LastOperation{
		State: gardencorev1beta1.LastOperationStateError,
	})
}

func fixShootInSucceededState(name string) *gardener_types.Shoot {
	return fixShoot(name, &gardener_types.LastOperation{
		State: gardencorev1beta1.LastOperationStateSucceeded,
	})
}

func fixShootInFailedState(name string) *gardener_types.Shoot {
	return fixShoot(name, &gardener_types.LastOperation{
		State: gardencorev1beta1.LastOperationStateFailed,
	})
}

func fixShootInUnknownState(name string) *gardener_types.Shoot {
	return fixShoot(name, nil)
}

func fixShoot(name string, lastOperation *gardener_types.LastOperation) *gardener_types.Shoot {
	return &gardener_types.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: gardener_types.ShootStatus{
			LastOperation: lastOperation,
		},
	}
}
