package update

import (
	"errors"
	"fmt"
	"time"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type CheckReconcilerState struct {
	operationManager *process.UpdateOperationManager
	reconcilerClient reconciler.Client
}

func NewCheckReconcilerState(os storage.Operations, reconcilerClient reconciler.Client) *CheckReconcilerState {
	return &CheckReconcilerState{
		operationManager: process.NewUpdateOperationManager(os),
		reconcilerClient: reconcilerClient,
	}
}

func (s *CheckReconcilerState) Name() string {
	return "CheckReconcilerState"
}

func (s *CheckReconcilerState) Run(operation internal.UpdatingOperation, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	state, err := s.reconcilerClient.GetCluster(operation.RuntimeID, operation.ClusterConfigurationVersion)

	if kebError.IsTemporaryError(err) {
		log.Errorf("Reconciler GetCluster method failed (temporary error, retrying): %v", err)
		return operation, 1 * time.Minute, nil
	} else if err != nil {
		return s.operationManager.OperationFailed(operation, err.Error(), err, log)
	}
	switch state.Status {
	case reconcilerApi.StatusReconciling, reconcilerApi.StatusReconcilePending:
		log.Infof("Reconciler status %v", state.Status)
		return operation, 30 * time.Second, nil
	case reconcilerApi.StatusReconcileErrorRetryable:
		log.Infof("Reconciler failed with retryable, rechecking in 10 minutes.")
		return operation, 10 * time.Minute, nil
	case reconcilerApi.StatusReady:
		return operation, 0, nil
	case reconcilerApi.StatusError:
		msg := fmt.Sprintf("Reconciler failed %v: %v", state.Status, reconciler.PrettyFailures(state))
		return s.operationManager.OperationFailed(operation, msg, errors.New(""), log)
	default:
		msg := fmt.Sprintf("Unknown reconciler cluster state %v, error: %v", state.Status, reconciler.PrettyFailures(state))
		return s.operationManager.OperationFailed(operation, msg, errors.New(""), log)
	}
}
