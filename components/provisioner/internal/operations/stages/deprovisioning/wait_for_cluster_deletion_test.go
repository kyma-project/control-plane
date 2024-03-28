package deprovisioning

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"

	"k8s.io/apimachinery/pkg/runtime/schema"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	gardener_mocks "github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/deprovisioning/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"
	dbMocks "github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

func TestWaitForClusterDeletion_Run(t *testing.T) {

	type mockFuncNoDirector func(gardenerClient *gardener_mocks.GardenerClient, dbSessionFactory *dbMocks.Factory)

	cluster := model.Cluster{
		ID: "runtimeID",
		ClusterConfig: model.GardenerConfig{
			Name: clusterName,
		},
		Tenant: "tenant",
	}

	for _, testCase := range []struct {
		description   string
		mockFunc      mockFuncNoDirector
		expectedStage model.OperationStage
		expectedDelay time.Duration
	}{
		{
			description: "should go to the next step when Shoot was deleted successfully",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSessionFactory *dbMocks.Factory) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, ""))
				dbSession := &dbMocks.WriteSessionWithinTransaction{}
				dbSession.On("MarkClusterAsDeleted", runtimeID).Return(nil)
				dbSessionFactory.On("NewSessionWithinTransaction").Return(dbSession, nil)

				dbSession.On("Commit").Return(nil)
				dbSession.On("RollbackUnlessCommitted").Return()
			},
			expectedStage: nextStageName,
			expectedDelay: 0,
		},
		{
			description: "should continue waiting if shoot not deleted",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSessionFactory *dbMocks.Factory) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(&gardener_types.Shoot{}, nil)
			},
			expectedStage: model.WaitForClusterDeletion,
			expectedDelay: 20 * time.Second,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardener_mocks.GardenerClient{}
			dbSessionFactory := &dbMocks.Factory{}

			testCase.mockFunc(gardenerClient, dbSessionFactory)

			waitForClusterDeletionStep := NewWaitForClusterDeletionStep(gardenerClient, dbSessionFactory, nextStageName, 10*time.Minute)

			// when
			result, err := waitForClusterDeletionStep.Run(cluster, model.Operation{}, logrus.New())

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedStage, result.Stage)
			assert.Equal(t, testCase.expectedDelay, result.Delay)
		})
	}

	for _, testCase := range []struct {
		description        string
		mockFunc           mockFuncNoDirector
		cluster            model.Cluster
		unrecoverableError bool
		errComponent       apperrors.ErrComponent
		errReason          apperrors.ErrReason
		errMsg             string
	}{
		{
			description: "should return error when failed to get shoot",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSessionFactory *dbMocks.Factory) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(nil, errors.New("some error"))
			},
			cluster:            cluster,
			unrecoverableError: false,
			errComponent:       apperrors.ErrGardenerClient,
			errReason:          "",
			errMsg:             "some error",
		},
		{
			description: "should return error when failed to start database transaction",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSessionFactory *dbMocks.Factory) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, ""))
				dbSessionFactory.On("NewSessionWithinTransaction").Return(nil, dberrors.Internal("some error"))
			},
			cluster:            cluster,
			unrecoverableError: false,
			errComponent:       apperrors.ErrDB,
			errReason:          dberrors.ErrDBInternal,
			errMsg:             "error starting db session with transaction: some error",
		},
		{
			description: "should return error when failed to mark cluster as deleted",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSessionFactory *dbMocks.Factory) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, ""))
				dbSession := &dbMocks.WriteSessionWithinTransaction{}
				dbSession.On("MarkClusterAsDeleted", runtimeID).Return(dberrors.NotFound("some error"))
				dbSessionFactory.On("NewSessionWithinTransaction").Return(dbSession, nil)
				dbSession.On("RollbackUnlessCommitted").Return()
			},
			cluster:            cluster,
			unrecoverableError: false,
			errComponent:       apperrors.ErrDB,
			errReason:          dberrors.ErrDBNotFound,
			errMsg:             "error marking cluster for deletion: some error",
		},
		{
			description: "should return error when failed to commit database transaction",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient, dbSessionFactory *dbMocks.Factory) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(nil, k8serrors.NewNotFound(schema.GroupResource{}, ""))
				dbSession := &dbMocks.WriteSessionWithinTransaction{}
				dbSession.On("MarkClusterAsDeleted", mock.AnythingOfType("string")).Return(nil)
				dbSessionFactory.On("NewSessionWithinTransaction").Return(dbSession, nil)

				dbSession.On("Commit").Return(dberrors.Internal("some error"))
				dbSession.On("RollbackUnlessCommitted").Return()
			},
			cluster:            cluster,
			unrecoverableError: false,
			errComponent:       apperrors.ErrDB,
			errReason:          dberrors.ErrDBInternal,
			errMsg:             "error commiting transaction: some error",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardener_mocks.GardenerClient{}
			dbSessionFactory := &dbMocks.Factory{}

			testCase.mockFunc(gardenerClient, dbSessionFactory)

			waitForClusterDeletionStep := NewWaitForClusterDeletionStep(gardenerClient, dbSessionFactory, nextStageName, 10*time.Minute)

			// when
			_, err := waitForClusterDeletionStep.Run(testCase.cluster, model.Operation{}, logrus.New())
			appErr := operations.ConvertToAppError(err)

			// then
			require.Error(t, err)
			nonRecoverable := operations.NonRecoverableError{}
			require.Equal(t, testCase.unrecoverableError, errors.As(err, &nonRecoverable))
			assert.Equal(t, testCase.errComponent, appErr.Component())
			assert.Equal(t, testCase.errReason, appErr.Reason())
			assert.Equal(t, testCase.errMsg, err.Error())
			gardenerClient.AssertExpectations(t)
			dbSessionFactory.AssertExpectations(t)
		})
	}

}
