package update

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"

	contract "github.com/kyma-incubator/reconciler/pkg/keb"
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
		return s.operationManager.OperationFailed(operation, err.Error(), log)
	}
	switch state.Status {
	case contract.StatusReconciling, contract.StatusReconcilePending, contract.StatusReconcileErrorRetryable:
		log.Info("Reconciler status %v", state.Status)
		return operation, 30 * time.Second, nil
	case contract.StatusReady:
		return operation, 0, nil
	case contract.StatusError:
		msg := fmt.Sprintf("Reconciler failed %v: %v", state.Status, reconciler.PrettyFailures(state))
		return s.operationManager.OperationFailed(operation, msg, log)
	default:
		msg := fmt.Sprintf("Unknown reconciler cluster state %v, error: %v", state.Status, reconciler.PrettyFailures(state))
		return s.operationManager.OperationFailed(operation, msg, log)
	}
}
