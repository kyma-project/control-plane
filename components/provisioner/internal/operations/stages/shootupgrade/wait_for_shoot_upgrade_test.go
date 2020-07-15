package shootupgrade

import (
	"errors"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	gardener_mocks "github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/deprovisioning/mocks"
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
		mockFunc      func(gardenerClient *gardener_mocks.GardenerClient)
		expectedStage model.OperationStage
		expectedDelay time.Duration
	}{
		{
			description: "should continue waiting if cluster is in processing state",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", clusterName, mock.Anything).Return(
					testkit.NewTestShoot(clusterName).
						WithOperationProcessing().
						ToShoot(), nil)
			},
			expectedStage: model.WaitingForShootUpgrade,
			expectedDelay: 20 * time.Second,
		},
		{
			description: "should continue waiting if cluster is in pending state",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", clusterName, mock.Anything).Return(
					testkit.NewTestShoot(clusterName).
						WithOperationPending().
						ToShoot(), nil)
			},
			expectedStage: model.WaitingForShootUpgrade,
			expectedDelay: 20 * time.Second,
		},
		{
			description: "should continue waiting if cluster is in error state - the operation will be retried on Gardener side",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", clusterName, mock.Anything).Return(
					testkit.NewTestShoot(clusterName).
						WithOperationError().
						ToShoot(), nil)
			},
			expectedStage: model.WaitingForShootUpgrade,
			expectedDelay: 20 * time.Second,
		},
		{
			description: "should continue waiting if last operation not set",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", clusterName, mock.Anything).Return(
					testkit.NewTestShoot(clusterName).
						WithOperationNil().
						ToShoot(), nil)
			},
			expectedStage: model.WaitingForShootUpgrade,
			expectedDelay: 20 * time.Second,
		},
		{
			description: "should return finished stage if cluster upgrade has succeeded",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {

				gardenerClient.On("Get", clusterName, mock.Anything).Return(
					testkit.NewTestShoot(clusterName).
						WithOperationSucceeded().
						ToShoot(), nil)
			},
			expectedStage: model.FinishedStage,
			expectedDelay: 0,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardener_mocks.GardenerClient{}

			testCase.mockFunc(gardenerClient)

			waitForrShootClusterUpgradeStep := NewWaitForShootUpgradeStep(gardenerClient, model.FinishedStage, time.Minute)
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
				gardenerClient.On("Get", clusterName, mock.Anything).Return(
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

			testCase.mockFunc(gardenerClient)

			waitForClusterCreationStep := NewWaitForShootUpgradeStep(gardenerClient, model.FinishedStage, time.Minute)

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
