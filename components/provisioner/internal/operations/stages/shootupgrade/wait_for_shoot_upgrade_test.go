package shootupgrade

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	gardener_mocks "github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/deprovisioning/mocks"
	shootupgrade_mocks "github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/shootupgrade/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"
	dbMocks "github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/testkit"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
		mockFunc      func(gardenerClient *gardener_mocks.GardenerClient, dbSession *dbMocks.ReadWriteSession, kubeconfigProvider *shootupgrade_mocks.KubeconfigProvider)
		expectedStage model.OperationStage
		expectedDelay time.Duration
	}{
		{
			description: "should continue waiting if cluster is in processing state",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSession *dbMocks.ReadWriteSession, kubeconfigProvider *shootupgrade_mocks.KubeconfigProvider) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(
					testkit.NewTestShoot(clusterName).
						WithOperationProcessing().
						ToShoot(), nil)
			},
			expectedStage: model.WaitingForShootUpgrade,
			expectedDelay: 20 * time.Second,
		},
		{
			description: "should continue waiting if cluster is in pending state",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSession *dbMocks.ReadWriteSession, kubeconfigProvider *shootupgrade_mocks.KubeconfigProvider) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(
					testkit.NewTestShoot(clusterName).
						WithOperationPending().
						ToShoot(), nil)
			},
			expectedStage: model.WaitingForShootUpgrade,
			expectedDelay: 20 * time.Second,
		},
		{
			description: "should continue waiting if cluster is in error state - the operation will be retried on Gardener side",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSession *dbMocks.ReadWriteSession, kubeconfigProvider *shootupgrade_mocks.KubeconfigProvider) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(
					testkit.NewTestShoot(clusterName).
						WithOperationError().
						ToShoot(), nil)
			},
			expectedStage: model.WaitingForShootUpgrade,
			expectedDelay: 20 * time.Second,
		},
		{
			description: "should continue waiting if last operation not set",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSession *dbMocks.ReadWriteSession, kubeconfigProvider *shootupgrade_mocks.KubeconfigProvider) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(
					testkit.NewTestShoot(clusterName).
						WithOperationNil().
						ToShoot(), nil)
			},
			expectedStage: model.WaitingForShootUpgrade,
			expectedDelay: 20 * time.Second,
		},
		{
			description: "should return finished stage if cluster upgrade has succeeded",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSession *dbMocks.ReadWriteSession, kubeconfigProvider *shootupgrade_mocks.KubeconfigProvider) {

				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(
					testkit.NewTestShoot(clusterName).
						WithOperationSucceeded().
						ToShoot(), nil)
				kubeconfigProvider.On("FetchFromShoot", clusterName).Return([]byte("kubeconfig"), nil)
				dbSession.On("UpdateKubeconfig", cluster.ID, "kubeconfig").Return(nil)
			},
			expectedStage: model.FinishedStage,
			expectedDelay: 0,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardener_mocks.GardenerClient{}
			dbSession := &dbMocks.ReadWriteSession{}
			kubeconfigProvider := &shootupgrade_mocks.KubeconfigProvider{}

			testCase.mockFunc(gardenerClient, dbSession, kubeconfigProvider)

			waitForShootClusterUpgradeStep := NewWaitForShootUpgradeStep(gardenerClient, dbSession, kubeconfigProvider, model.FinishedStage, time.Minute)
			// when
			result, err := waitForShootClusterUpgradeStep.Run(cluster, model.Operation{}, logrus.New())

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedStage, result.Stage)
			assert.Equal(t, testCase.expectedDelay, result.Delay)
			gardenerClient.AssertExpectations(t)
		})
	}

	for _, testCase := range []struct {
		description        string
		mockFunc           func(gardenerClient *gardener_mocks.GardenerClient, dbSession *dbMocks.ReadWriteSession, kubeconfigProvider *shootupgrade_mocks.KubeconfigProvider)
		unrecoverableError bool
		cluster            model.Cluster
	}{
		{
			description: "should return error if failed to read Shoot",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSession *dbMocks.ReadWriteSession, kubeconfigProvider *shootupgrade_mocks.KubeconfigProvider) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(nil, errors.New("some error"))
			},
			unrecoverableError: false,
			cluster:            cluster,
		},
		{
			description: "should return error if Shoot is in failed state with rate limit exceeded error",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSession *dbMocks.ReadWriteSession, kubeconfigProvider *shootupgrade_mocks.KubeconfigProvider) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(
					testkit.NewTestShoot(clusterName).
						WithOperationFailed().
						WithRateLimitExceededError().
						ToShoot(), nil)
			},
			unrecoverableError: false,
			cluster:            cluster,
		},
		{
			description: "should return an error if failed to update encrypted kubeconfig",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSession *dbMocks.ReadWriteSession, kubeconfigProvider *shootupgrade_mocks.KubeconfigProvider) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(
					testkit.NewTestShoot(clusterName).
						WithOperationSucceeded().
						ToShoot(), nil)
				kubeconfigProvider.On("FetchFromShoot", clusterName).Return([]byte("kubeconfig"), nil)
				dbSession.On("UpdateKubeconfig", cluster.ID, "kubeconfig").Return(dberrors.Internal("error"))
			},
			unrecoverableError: false,
			cluster:            cluster,
		},
		{
			description: "should return unrecoverable error if Shoot is in failed state",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSession *dbMocks.ReadWriteSession, kubeconfigProvider *shootupgrade_mocks.KubeconfigProvider) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(
					testkit.NewTestShoot(clusterName).
						WithOperationFailed().
						ToShoot(), nil)
			},
			unrecoverableError: true,
			cluster:            cluster,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardener_mocks.GardenerClient{}
			dbSession := &dbMocks.ReadWriteSession{}
			kubeconfigProvider := &shootupgrade_mocks.KubeconfigProvider{}

			testCase.mockFunc(gardenerClient, dbSession, kubeconfigProvider)

			waitForClusterCreationStep := NewWaitForShootUpgradeStep(gardenerClient, dbSession, kubeconfigProvider, model.FinishedStage, time.Minute)

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
