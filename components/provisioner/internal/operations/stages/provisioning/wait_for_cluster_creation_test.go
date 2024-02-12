package provisioning

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	gardener_mocks "github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/deprovisioning/mocks"
	provisioning_mocks "github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/provisioning/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"
	dbMocks "github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWaitForClusterInitialization_Run(t *testing.T) {

	clusterName := "name"
	runtimeID := "runtimeID"
	tenant := "tenant"

	cluster := model.Cluster{
		ID:     runtimeID,
		Tenant: tenant,
		ClusterConfig: model.GardenerConfig{
			Name: clusterName,
			Seed: "az-eu2",
		},
	}

	clusterWithoutSeed := model.Cluster{
		ID:     runtimeID,
		Tenant: tenant,
		ClusterConfig: model.GardenerConfig{
			Name: clusterName,
		},
	}

	for _, testCase := range []struct {
		description   string
		mockFunc      func(ardenerClient *gardener_mocks.GardenerClient, dbSession *dbMocks.ReadWriteSession, kubeconfigProvider *provisioning_mocks.DynamicKubeconfigProvider)
		expectedStage model.OperationStage
		expectedDelay time.Duration
		cluster       model.Cluster
	}{
		{
			description: "should continue waiting if cluster not created",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, _ *dbMocks.ReadWriteSession, _ *provisioning_mocks.DynamicKubeconfigProvider) {

				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(fixShootInProcessingState(clusterName), nil)
			},
			expectedStage: model.WaitingForClusterCreation,
			expectedDelay: 20 * time.Second,
			cluster:       cluster,
		},
		{
			description: "should continue waiting if last operation not set",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, _ *dbMocks.ReadWriteSession, _ *provisioning_mocks.DynamicKubeconfigProvider) {

				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(fixShootInUnknownState(clusterName), nil)
			},
			expectedStage: model.WaitingForClusterCreation,
			expectedDelay: 20 * time.Second,
			cluster:       cluster,
		},
		{
			description: "should go to the next stage if cluster was created based on configuration with gardener seed provided",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSession *dbMocks.ReadWriteSession, _ *provisioning_mocks.DynamicKubeconfigProvider) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(fixShootInSucceededStateWithSeed(clusterName, "az-eu2"), nil)
				dbSession.On("UpdateKubeconfig", cluster.ID, "kubeconfig").Return(nil)
			},
			expectedStage: nextStageName,
			expectedDelay: 0,
			cluster:       cluster,
		},
		{
			description: "should go to the next stage if cluster was created based on configuration without gardener seed provided",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSession *dbMocks.ReadWriteSession, _ *provisioning_mocks.DynamicKubeconfigProvider) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(fixShootInSucceededStateWithSeed(clusterName, "az-eu2"), nil)

				dbSession.On("UpdateKubeconfig", cluster.ID, "kubeconfig").Return(nil)
				dbSession.On("UpdateGardenerClusterConfig", cluster.ClusterConfig).Return(nil)
			},
			expectedStage: nextStageName,
			expectedDelay: 0,
			cluster:       clusterWithoutSeed,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardener_mocks.GardenerClient{}
			dbSession := &dbMocks.ReadWriteSession{}
			kubeconfigProvider := &provisioning_mocks.DynamicKubeconfigProvider{}

			testCase.mockFunc(gardenerClient, dbSession, kubeconfigProvider)

			waitForClusterCreationStep := NewWaitForClusterCreationStep(gardenerClient, dbSession, nextStageName, 10*time.Minute)
			// when
			result, err := waitForClusterCreationStep.Run(testCase.cluster, model.Operation{}, logrus.New())

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedStage, result.Stage)
			assert.Equal(t, testCase.expectedDelay, result.Delay)
			gardenerClient.AssertExpectations(t)
		})
	}

	for _, testCase := range []struct {
		description        string
		mockFunc           func(gardenerClient *gardener_mocks.GardenerClient, dbSession *dbMocks.ReadWriteSession, kubeconfigProvider *provisioning_mocks.DynamicKubeconfigProvider)
		cluster            model.Cluster
		unrecoverableError bool
	}{
		{
			description: "should return error if failed to read Shoot",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, _ *dbMocks.ReadWriteSession, _ *provisioning_mocks.DynamicKubeconfigProvider) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(nil, errors.New("some error"))
			},
			unrecoverableError: false,
			cluster:            cluster,
		},
		{
			description: "should return error if Shoot is in failed state due to rate limits exceeded",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, _ *dbMocks.ReadWriteSession, _ *provisioning_mocks.DynamicKubeconfigProvider) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(fixShootInFailedStateWithLimitRatingError(clusterName), nil)
			},
			unrecoverableError: false,
			cluster:            cluster,
		},
		{
			description: "should return error if Shoot is in failed state during reconcile operation",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, _ *dbMocks.ReadWriteSession, _ *provisioning_mocks.DynamicKubeconfigProvider) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(fixShootDuringReconcileInFailedState(clusterName), nil)
			},
			unrecoverableError: false,
			cluster:            cluster,
		},
		{
			description: "should return unrecoverable error if Shoot is in failed state",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, _ *dbMocks.ReadWriteSession, _ *provisioning_mocks.DynamicKubeconfigProvider) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(fixShootInFailedState(clusterName), nil)
			},
			unrecoverableError: true,
			cluster:            cluster,
		},
		{
			description: "should return error if failed to update seed in database",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSession *dbMocks.ReadWriteSession, _ *provisioning_mocks.DynamicKubeconfigProvider) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(fixShootInSucceededStateWithSeed(clusterName, "az-eu2"), nil)

				dbSession.On("UpdateGardenerClusterConfig", cluster.ClusterConfig).Return(dberrors.Internal("some error"))
			},
			unrecoverableError: false,
			cluster:            clusterWithoutSeed,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardener_mocks.GardenerClient{}
			dbSession := &dbMocks.ReadWriteSession{}
			kubeconfigProvider := &provisioning_mocks.DynamicKubeconfigProvider{}

			testCase.mockFunc(gardenerClient, dbSession, kubeconfigProvider)

			waitForClusterCreationStep := NewWaitForClusterCreationStep(gardenerClient, dbSession, nextStageName, 10*time.Minute)

			// when
			_, err := waitForClusterCreationStep.Run(testCase.cluster, model.Operation{}, logrus.New())

			// then
			require.Error(t, err)
			nonRecoverable := operations.NonRecoverableError{}
			require.Equal(t, testCase.unrecoverableError, errors.As(err, &nonRecoverable))

			gardenerClient.AssertExpectations(t)
			dbSession.AssertExpectations(t)
			kubeconfigProvider.AssertExpectations(t)
		})
	}
}

func fixShootInSucceededState(name string) *v1beta1.Shoot {
	return fixShoot(name, &v1beta1.LastOperation{
		State: v1beta1.LastOperationStateSucceeded,
	})
}

func fixShootInSucceededStateWithSeed(name string, seed string) *v1beta1.Shoot {
	shoot := fixShootInSucceededState(name)
	shoot.Spec.SeedName = &seed
	return shoot
}

func fixShootInFailedState(name string) *v1beta1.Shoot {
	return fixShoot(name, &v1beta1.LastOperation{
		State: v1beta1.LastOperationStateFailed,
	})
}

func fixShootDuringReconcileInFailedState(name string) *v1beta1.Shoot {
	return fixShoot(name, &v1beta1.LastOperation{
		State: v1beta1.LastOperationStateFailed,
		Type:  v1beta1.LastOperationTypeReconcile,
	})
}

func fixShootInFailedStateWithLimitRatingError(name string) *v1beta1.Shoot {
	shoot := fixShoot(name, &v1beta1.LastOperation{
		State: v1beta1.LastOperationStateFailed,
	})

	codes := make([]v1beta1.ErrorCode, 1)
	codes[0] = v1beta1.ErrorInfraRateLimitsExceeded

	lastError := v1beta1.LastError{Codes: codes}

	lastErrors := make([]v1beta1.LastError, 1)
	lastErrors[0] = lastError
	shoot.Status.LastErrors = lastErrors

	return shoot
}

func fixShootInProcessingState(name string) *v1beta1.Shoot {
	return fixShoot(name, &v1beta1.LastOperation{
		State: v1beta1.LastOperationStateProcessing,
	})
}

func fixShootInUnknownState(name string) *v1beta1.Shoot {
	return fixShoot(name, nil)
}

func fixShoot(name string, lastOperation *v1beta1.LastOperation) *v1beta1.Shoot {
	return &v1beta1.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: v1beta1.ShootStatus{
			LastOperation: lastOperation,
		},
	}
}
