package deprovisioning

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/util/testkit"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"

	installationMocks "github.com/kyma-project/control-plane/components/provisioner/internal/installation/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	gardener_mocks "github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/deprovisioning/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const gardenerNamespace = "default"

func TestCleanupCluster_Run(t *testing.T) {

	clusterWithKubeconfig := model.Cluster{
		ClusterConfig: model.GardenerConfig{
			Name: clusterName,
		},
		Kubeconfig: util.StringPtr(kubeconfig),
	}

	clusterWithoutKubeconfig := model.Cluster{
		ClusterConfig: model.GardenerConfig{
			Name: clusterName,
		},
	}

	clusterWithInvalidKubeconfig := model.Cluster{
		ClusterConfig: model.GardenerConfig{
			Name: clusterName,
		},
		Kubeconfig: util.StringPtr("invalid"),
	}

	for _, testCase := range []struct {
		description   string
		mockFunc      func(gardenerClient *gardener_mocks.GardenerClient, installationSvc *installationMocks.Service)
		expectedStage model.OperationStage
		expectedDelay time.Duration
		cluster       model.Cluster
	}{
		{
			description: "should go to the next step when kubeconfig is empty",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, installationSvc *installationMocks.Service) {
			},
			expectedStage: nextStageName,
			expectedDelay: 0,
			cluster:       clusterWithoutKubeconfig,
		},
		{
			description: "should go to the next step when cluster is hibernated",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, installationSvc *installationMocks.Service) {
				shoot := testkit.NewTestShoot(clusterName).
					InNamespace(gardenerNamespace).
					WithHibernationState(true, true).
					ToShoot()

				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(shoot, nil)
			},
			expectedStage: nextStageName,
			expectedDelay: 0,
			cluster:       clusterWithKubeconfig,
		},
		{
			description: "should go to the next step when cleanup was performed successfully",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, installationSvc *installationMocks.Service) {
				shoot := testkit.NewTestShoot(clusterName).
					InNamespace(gardenerNamespace).
					WithHibernationState(true, false).
					ToShoot()

				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(shoot, nil)
				installationSvc.On("PerformCleanup", mock.AnythingOfType("*rest.Config")).Return(nil)
			},
			expectedStage: nextStageName,
			expectedDelay: 0,
			cluster:       clusterWithKubeconfig,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			installationSvc := &installationMocks.Service{}
			gardenerClient := &gardener_mocks.GardenerClient{}

			testCase.mockFunc(gardenerClient, installationSvc)

			cleanupClusterStep := NewCleanupClusterStep(gardenerClient, installationSvc, nextStageName, 10*time.Minute)

			// when
			result, err := cleanupClusterStep.Run(testCase.cluster, model.Operation{}, logrus.New())

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedStage, result.Stage)
			assert.Equal(t, testCase.expectedDelay, result.Delay)
			installationSvc.AssertExpectations(t)
			gardenerClient.AssertExpectations(t)
		})
	}

	for _, testCase := range []struct {
		description        string
		mockFunc           func(gardenerClient *gardener_mocks.GardenerClient, installationSvc *installationMocks.Service)
		cluster            model.Cluster
		unrecoverableError bool
		errComponent       apperrors.ErrComponent
		errReason          apperrors.ErrReason
		errMsg             string
	}{
		{
			description: "should return error if failed to get shoot",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, installationSvc *installationMocks.Service) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(nil, errors.New("some error"))
			},
			cluster:            clusterWithKubeconfig,
			unrecoverableError: false,
			errComponent:       apperrors.ErrGardenerClient,
			errReason:          "",
			errMsg:             "some error",
		},
		{
			description: "should return error is failed to parse kubeconfig",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, installationSvc *installationMocks.Service) {
				shoot := testkit.NewTestShoot(clusterName).
					InNamespace(gardenerNamespace).
					WithHibernationState(true, false).
					ToShoot()

				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(shoot, nil)
			},
			cluster:            clusterWithInvalidKubeconfig,
			unrecoverableError: true,
			errComponent:       apperrors.ErrClusterK8SClient,
			errReason:          "",
			errMsg:             "error: failed to create kubernetes config from raw: error constructing kubeconfig from raw config: ",
		},
		{
			description: "should return error when failed to perform cleanup",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, installationSvc *installationMocks.Service) {
				shoot := testkit.NewTestShoot(clusterName).
					InNamespace(gardenerNamespace).
					WithHibernationState(true, false).
					ToShoot()

				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(shoot, nil)
				installationSvc.On("PerformCleanup", mock.AnythingOfType("*rest.Config")).Return(util.K8SErrorToAppError(errors.New("timeout")))
			},
			cluster:            clusterWithKubeconfig,
			unrecoverableError: false,
			errComponent:       apperrors.ErrClusterK8SClient,
			errReason:          "",
			errMsg:             "timeout",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			installationSvc := &installationMocks.Service{}
			gardenerClient := &gardener_mocks.GardenerClient{}

			testCase.mockFunc(gardenerClient, installationSvc)

			cleanupClusterStep := NewCleanupClusterStep(gardenerClient, installationSvc, nextStageName, 10*time.Minute)

			// when
			_, err := cleanupClusterStep.Run(testCase.cluster, model.Operation{}, logrus.New())
			appErr := operations.ConvertToAppError(err)

			// then
			require.Error(t, err)
			nonRecoverable := operations.NonRecoverableError{}
			require.Equal(t, testCase.unrecoverableError, errors.As(err, &nonRecoverable))
			assert.Equal(t, testCase.errComponent, appErr.Component())
			assert.Equal(t, testCase.errReason, appErr.Reason())
			assert.Contains(t, err.Error(), testCase.errMsg)
			installationSvc.AssertExpectations(t)
			gardenerClient.AssertExpectations(t)
		})
	}
}
