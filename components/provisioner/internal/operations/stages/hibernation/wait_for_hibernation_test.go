package hibernation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/hibernation/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/testkit"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWaitForHibernation(t *testing.T) {

	const (
		nextStageName = model.FinishedStage
		clusterName   = "test"
	)

	runtimeID := "runtimeID"

	cluster := model.Cluster{
		ID: runtimeID,
		ClusterConfig: model.GardenerConfig{
			Name: clusterName,
		},
	}

	for _, testCase := range []struct {
		description   string
		mockFunc      func(gardenerClient *mocks.GardenerClient)
		expectedStage model.OperationStage
		expectedDelay time.Duration
	}{
		{
			description: "should wait if cluster not hibernated",
			mockFunc: func(gardenerClient *mocks.GardenerClient) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(
					testkit.NewTestShoot(clusterName).
						WithHibernationState(true, false).
						ToShoot(), nil)
			},
			expectedStage: model.WaitForHibernation,
			expectedDelay: 30 * time.Second,
		},
		{
			description: "should go to the next state if cluster is hibernated",
			mockFunc: func(gardenerClient *mocks.GardenerClient) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(testkit.NewTestShoot(clusterName).
					WithHibernationState(true, true).
					ToShoot(), nil)
			},
			expectedStage: nextStageName,
			expectedDelay: 0,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &mocks.GardenerClient{}

			testCase.mockFunc(gardenerClient)

			checkHibernationConditionStep := NewWaitForHibernationStep(gardenerClient, nextStageName, time.Minute)

			// when
			result, err := checkHibernationConditionStep.Run(cluster, model.Operation{}, logrus.New())

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedStage, result.Stage)
			assert.Equal(t, testCase.expectedDelay, result.Delay)
			gardenerClient.AssertExpectations(t)
		})
	}

	for _, testCase := range []struct {
		description        string
		mockFunc           func(gardenerClient *mocks.GardenerClient)
		unrecoverableError bool
	}{
		{
			description: "should return error if failed to get shoot",
			mockFunc: func(gardenerClient *mocks.GardenerClient) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(
					nil, errors.New("some error"))
			},
			unrecoverableError: false,
		},
		{
			description: "should return unrecoverable error when last operation failed",
			mockFunc: func(gardenerClient *mocks.GardenerClient) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(testkit.NewTestShoot(clusterName).
					WithOperationFailed().
					ToShoot(), nil)
			},
			unrecoverableError: true,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &mocks.GardenerClient{}

			testCase.mockFunc(gardenerClient)

			checkHibernationConditionStep := NewWaitForHibernationStep(gardenerClient, nextStageName, time.Minute)

			// when
			_, err := checkHibernationConditionStep.Run(cluster, model.Operation{}, logrus.New())

			// then
			require.Error(t, err)
			nonRecoverable := operations.NonRecoverableError{}
			require.Equal(t, testCase.unrecoverableError, errors.As(err, &nonRecoverable))
			gardenerClient.AssertExpectations(t)
		})
	}
}
