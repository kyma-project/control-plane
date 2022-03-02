package operations

import (
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"

	"github.com/kyma-incubator/compass/components/director/pkg/graphql"

	directorMocks "github.com/kyma-project/control-plane/components/provisioner/internal/director/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/failure"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	operationId = "operation-id"
	clusterId   = "cluster-id"
)

func TestStagesExecutor_Execute(t *testing.T) {

	tNow := time.Now()

	operation := model.Operation{
		ID:             operationId,
		Type:           model.Provision,
		StartTimestamp: tNow,
		EndTimestamp:   nil,
		State:          model.InProgress,
		ClusterID:      clusterId,
		Stage:          model.WaitingForInstallation,
		LastTransition: &tNow,
	}

	cluster := model.Cluster{ID: clusterId}

	t.Run("should not requeue operation when stage if Finished", func(t *testing.T) {
		// given
		dbSession := &mocks.ReadWriteSession{}
		dbSession.On("GetOperation", operationId).Return(operation, nil)
		dbSession.On("GetCluster", clusterId).Return(cluster, nil)
		dbSession.On("TransitionOperation", operationId, "Provisioning steps finished", model.FinishedStage, mock.AnythingOfType("time.Time")).
			Return(nil)
		dbSession.On("UpdateOperationState", operationId, "Operation succeeded", model.Succeeded, mock.AnythingOfType("time.Time")).
			Return(nil)
		dbSession.On("UpdateOperationLastError", operationId, "", "", "").Return(nil)

		mockStage := NewMockStep(model.WaitingForInstallation, model.FinishedStage, 10*time.Second, 10*time.Second)

		installationStages := map[model.OperationStage]Step{
			model.WaitingForInstallation: mockStage,
		}

		directorClient := &directorMocks.DirectorClient{}

		executor := NewExecutor(dbSession, model.Provision, installationStages, failure.NewNoopFailureHandler(), directorClient)

		// when
		result := executor.Execute(operationId)

		// then
		assert.Equal(t, false, result.Requeue)
		assert.True(t, mockStage.called)
	})

	t.Run("should requeue operation if error occurred", func(t *testing.T) {
		// given
		runErr := fmt.Errorf("error")
		dbSession := &mocks.ReadWriteSession{}
		dbSession.On("GetOperation", operationId).Return(operation, nil)
		dbSession.On("GetCluster", clusterId).Return(cluster, nil)
		dbSession.On("UpdateOperationLastError", operationId, runErr.Error(), string(apperrors.ErrProvisionerInternal), string(apperrors.ErrProvisioner)).Return(nil)

		mockStage := NewErrorStep(model.WaitingForClusterCreation, runErr, time.Second*10)

		installationStages := map[model.OperationStage]Step{
			model.WaitingForInstallation: mockStage,
		}

		directorClient := &directorMocks.DirectorClient{}

		executor := NewExecutor(dbSession, model.Provision, installationStages, failure.NewNoopFailureHandler(), directorClient)

		// when
		result := executor.Execute(operationId)

		// then
		assert.Equal(t, true, result.Requeue)
		assert.True(t, mockStage.called)
	})

	t.Run("should not requeue operation and run failure handler if NonRecoverable error occurred", func(t *testing.T) {
		// given
		runErr := NewNonRecoverableError(apperrors.External("gardener error").SetComponent(apperrors.ErrGardener).SetReason("ERR_INFRA_QUOTA_EXCEEDED").Append("something"))
		dbSession := &mocks.ReadWriteSession{}
		dbSession.On("GetOperation", operationId).Return(operation, nil)
		dbSession.On("GetCluster", clusterId).Return(cluster, nil)
		dbSession.On("UpdateOperationState", operationId, "something, gardener error", model.Failed, mock.AnythingOfType("time.Time")).
			Return(nil)
		dbSession.On("UpdateOperationLastError", operationId, "something, gardener error", "ERR_INFRA_QUOTA_EXCEEDED", string(apperrors.ErrGardener)).Return(nil)

		mockStage := NewErrorStep(model.WaitingForClusterCreation, runErr, 10*time.Second)

		installationStages := map[model.OperationStage]Step{
			model.WaitingForInstallation: mockStage,
		}

		directorClient := &directorMocks.DirectorClient{}
		directorClient.On("SetRuntimeStatusCondition", clusterId, graphql.RuntimeStatusConditionFailed, mock.AnythingOfType("string")).Return(nil)

		failureHandler := MockFailureHandler{}

		executor := NewExecutor(dbSession, model.Provision, installationStages, &failureHandler, directorClient)

		// when
		result := executor.Execute(operationId)

		// then
		assert.Equal(t, false, result.Requeue)
		assert.True(t, mockStage.called)
		assert.True(t, failureHandler.called)
	})

	t.Run("should not requeue operation and run failure handler if NonRecoverable error occurred but failed to update Director", func(t *testing.T) {
		// given
		runErr := NewNonRecoverableError(apperrors.External(errors.Wrap(fmt.Errorf("error"), "kyma installation").Error()).SetComponent(apperrors.ErrKymaInstaller).SetReason("istio"))
		dbSession := &mocks.ReadWriteSession{}
		dbSession.On("GetOperation", operationId).Return(operation, nil)
		dbSession.On("GetCluster", clusterId).Return(cluster, nil)
		dbSession.On("UpdateOperationState", operationId, "kyma installation: error", model.Failed, mock.AnythingOfType("time.Time")).
			Return(nil)
		dbSession.On("UpdateOperationLastError", operationId, "kyma installation: error", "istio", string(apperrors.ErrKymaInstaller)).Return(nil)

		mockStage := NewErrorStep(model.StartingInstallation, runErr, 10*time.Second)

		installationStages := map[model.OperationStage]Step{
			model.WaitingForInstallation: mockStage,
		}

		directorClient := &directorMocks.DirectorClient{}
		directorClient.On("SetRuntimeStatusCondition", clusterId, graphql.RuntimeStatusConditionFailed, mock.AnythingOfType("string")).Return(apperrors.Internal("error"))

		failureHandler := MockFailureHandler{}

		executor := NewExecutor(dbSession, model.Provision, installationStages, &failureHandler, directorClient)

		// when
		result := executor.Execute(operationId)

		// then
		assert.False(t, result.Requeue)
		assert.True(t, mockStage.called)
		assert.True(t, failureHandler.called)
	})

	t.Run("should not requeue operation and run failure handler if timeout reached", func(t *testing.T) {
		// given
		dbSession := &mocks.ReadWriteSession{}
		dbSession.On("GetOperation", operationId).Return(operation, nil)
		dbSession.On("GetCluster", clusterId).Return(cluster, nil)
		dbSession.On("TransitionOperation", operationId, "Operation in progress", model.ConnectRuntimeAgent, mock.AnythingOfType("time.Time")).
			Return(nil)
		dbSession.On("UpdateOperationState", operationId, "error: timeout while processing operation", model.Failed, mock.AnythingOfType("time.Time")).
			Return(nil)

		mockStage := NewMockStep(model.WaitingForInstallation, model.ConnectRuntimeAgent, 0, 0*time.Second)

		installationStages := map[model.OperationStage]Step{
			model.WaitingForInstallation: mockStage,
		}

		directorClient := &directorMocks.DirectorClient{}
		directorClient.On("SetRuntimeStatusCondition", clusterId, graphql.RuntimeStatusConditionFailed, mock.AnythingOfType("string")).Return(nil)

		failureHandler := MockFailureHandler{}

		executor := NewExecutor(dbSession, model.Provision, installationStages, &failureHandler, directorClient)

		// when
		result := executor.Execute(operationId)

		// then
		assert.Equal(t, false, result.Requeue)
		assert.False(t, mockStage.called)
		assert.True(t, failureHandler.called)
	})
}

type mockStep struct {
	name      model.OperationStage
	next      model.OperationStage
	delay     time.Duration
	timeLimit time.Duration
	err       error

	called bool
}

func NewMockStep(name, next model.OperationStage, delay time.Duration, timeLimit time.Duration) *mockStep {
	return &mockStep{
		name:      name,
		next:      next,
		delay:     delay,
		timeLimit: timeLimit,
	}
}

func NewErrorStep(name model.OperationStage, err error, timeLimit time.Duration) *mockStep {
	return &mockStep{
		name:      name,
		err:       err,
		timeLimit: timeLimit,
	}
}

func (m mockStep) Name() model.OperationStage {
	return m.name
}

func (m *mockStep) Run(cluster model.Cluster, operation model.Operation, logger logrus.FieldLogger) (StageResult, error) {

	m.called = true

	if m.err != nil {
		return StageResult{}, m.err
	}

	return StageResult{
		Stage: m.next,
		Delay: m.delay,
	}, nil
}

func (m mockStep) TimeLimit() time.Duration {
	return m.timeLimit
}

type MockFailureHandler struct {
	called bool
}

func (m *MockFailureHandler) HandleFailure(operation model.Operation, cluster model.Cluster) error {
	m.called = true
	return nil
}

func TestConvertToAppError(t *testing.T) {
	t.Run("should convert to app error", func(t *testing.T) {
		//given
		err1 := fmt.Errorf("err1")
		err2 := apperrors.Internal("err2")
		DbErr := dberrors.NotFound("db error")
		nonRecoverableErr1 := NewNonRecoverableError(err1)
		nonRecoverableErr2 := NewNonRecoverableError(err2)
		k8sErr := util.K8SErrorToAppError(errors.Wrapf(err1, "failed to create %s ClusterRoleBinding", "crb.Name")).SetComponent(apperrors.ErrClusterK8SClient)

		expectErr1 := apperrors.Internal("err1")
		expectK8sErr := apperrors.Internal(errors.Wrapf(err1, "failed to create %s ClusterRoleBinding", "crb.Name").Error()).SetComponent(apperrors.ErrClusterK8SClient)

		//when
		apperr1 := ConvertToAppError(err1)
		apperr2 := ConvertToAppError(err2)
		apperrDbErr := ConvertToAppError(DbErr)
		apperrNonRecovErr1 := ConvertToAppError(nonRecoverableErr1)
		apperrNonRecovErr2 := ConvertToAppError(nonRecoverableErr2)
		apperrK8sErr := ConvertToAppError(k8sErr)

		//then
		assert.Equal(t, expectErr1, apperr1)
		assert.Equal(t, err2, apperr2)
		assert.Equal(t, DbErr, apperrDbErr)
		assert.Equal(t, apperrors.ErrDB, apperrDbErr.Component())
		assert.Equal(t, dberrors.ErrDBNotFound, apperrDbErr.Reason())
		assert.Equal(t, expectErr1, apperrNonRecovErr1)
		assert.Equal(t, err2, apperrNonRecovErr2)
		assert.Equal(t, expectK8sErr, apperrK8sErr)
	})
}
