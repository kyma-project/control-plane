package deprovisioning

import (
	"context"
	"errors"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	directorMocks "github.com/kyma-project/control-plane/components/provisioner/internal/director/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	gardener_mocks "github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/deprovisioning/mocks"
	dbMocks "github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	runtimeID                          = "runtimeID"
	tenant                             = "tenant"
	clusterName                        = "my cluster"
	nextStageName model.OperationStage = "NextStage"
)

func TestDeprovisionCluster_Run(t *testing.T) {

	cluster := model.Cluster{
		ID: "runtimeID",
		ClusterConfig: model.GardenerConfig{
			Name: clusterName,
		},
		Tenant: "tenant",
	}

	for _, testCase := range []struct {
		description   string
		mockFunc      func(gardenerClient *gardener_mocks.GardenerClient)
		expectedStage model.OperationStage
		expectedDelay time.Duration
	}{
		{
			description: "should go to the next step when Shoot was deleted successfully",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Delete", context.Background(), clusterName, mock.Anything).Return(nil)
			},
			expectedStage: nextStageName,
			expectedDelay: 0,
		},
		{
			description: "should go to the next step when Shoot not exists",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Delete", context.Background(), clusterName, mock.Anything).Return(k8serrors.NewNotFound(schema.GroupResource{}, ""))
			},
			expectedStage: nextStageName,
			expectedDelay: 0,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardener_mocks.GardenerClient{}

			testCase.mockFunc(gardenerClient)

			deleteClusterStep := NewDeleteClusterStep(gardenerClient, nextStageName, 10*time.Minute)

			// when
			result, err := deleteClusterStep.Run(cluster, model.Operation{}, logrus.New())

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
		cluster            model.Cluster
		unrecoverableError bool
		errComponent       apperrors.ErrComponent
		errReason          apperrors.ErrReason
		errMsg             string
	}{
		{
			description: "should return error when failed to delete shoot",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Delete", context.Background(), clusterName, mock.Anything).Return(k8serrors.NewBadRequest("some error"))
			},
			cluster:            cluster,
			unrecoverableError: false,
			errComponent:       apperrors.ErrGardenerClient,
			errReason:          apperrors.ErrReason("BadRequest"),
			errMsg:             "some error",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardener_mocks.GardenerClient{}
			dbSessionFactory := &dbMocks.Factory{}
			directorClient := &directorMocks.DirectorClient{}

			testCase.mockFunc(gardenerClient)

			deleteClusterStep := NewDeleteClusterStep(gardenerClient, nextStageName, 10*time.Minute)

			// when
			_, err := deleteClusterStep.Run(testCase.cluster, model.Operation{}, logrus.New())
			appErr := operations.ConvertToAppError(err)

			// then
			require.Error(t, err)
			nonRecoverable := operations.NonRecoverableError{}
			require.Equal(t, testCase.unrecoverableError, errors.As(err, &nonRecoverable))
			assert.Equal(t, testCase.errComponent, appErr.Component())
			assert.Equal(t, testCase.errReason, appErr.Reason())
			assert.Error(t, err, testCase.errMsg)
			gardenerClient.AssertExpectations(t)
			dbSessionFactory.AssertExpectations(t)
			directorClient.AssertExpectations(t)
		})
	}
}
