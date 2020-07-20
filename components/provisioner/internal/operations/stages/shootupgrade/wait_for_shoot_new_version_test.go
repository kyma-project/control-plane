package shootupgrade

import (
	"errors"
	"testing"
	"time"

	gardener_mocks "github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/deprovisioning/mocks"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/testkit"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWaitForNewShootVersion(t *testing.T) {
	clusterName := "shootName"
	runtimeID := "runtimeID"
	tenant := "tenant"
	operationID := "operationID"

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
			description: "should continue waiting if cluster has old resource version",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", clusterName, mock.Anything).Return(
					testkit.NewTestShoot(clusterName).
						WithGeneration(2).
						WithObservedGeneration(1).
						ToShoot(), nil)
			},
			expectedStage: model.WaitingForShootNewVersion,
			expectedDelay: 5 * time.Second,
		},
		{
			description: "should move to next step if resource version changes",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", clusterName, mock.Anything).Return(
					testkit.NewTestShoot(clusterName).
						WithGeneration(2).
						WithObservedGeneration(2).
						ToShoot(), nil)
			},
			expectedStage: model.WaitingForShootUpgrade,
			expectedDelay: 0,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardener_mocks.GardenerClient{}

			testCase.mockFunc(gardenerClient)

			waitForShootClusterUpgradeStep := NewWaitForShootNewVersionStep(gardenerClient, model.WaitingForShootUpgrade, time.Minute)

			// when
			result, err := waitForShootClusterUpgradeStep.Run(cluster, model.Operation{ID: operationID}, logrus.New())

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
						WithGeneration(2).
						WithObservedGeneration(1).
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

			waitForClusterCreationStep := NewWaitForShootNewVersionStep(gardenerClient, model.FinishedStage, time.Minute)

			// when
			_, err := waitForClusterCreationStep.Run(testCase.cluster, model.Operation{}, logrus.New())

			// then
			require.Error(t, err)
			require.Equal(t, testCase.unrecoverableError, errors.As(err, &operations.NonRecoverableError{}))
			gardenerClient.AssertExpectations(t)
		})
	}
}
