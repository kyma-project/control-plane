package update

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type LastRuntimeStep struct {
	operationManager *process.UpdateOperationManager
	runtimeStatesDb  storage.RuntimeStates
}

func NewLastRuntimeStep(os storage.Operations, runtimeStatesDb storage.RuntimeStates) *InitKymaVersionStep {
	return &InitKymaVersionStep{
		operationManager: process.NewUpdateOperationManager(os),
		runtimeStatesDb:  runtimeStatesDb,
	}
}

func (s *LastRuntimeStep) Name() string {
	return "Update_Last_Runtime_Step"
}

func (s *LastRuntimeStep) Run(operation internal.UpdatingOperation, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	var lrs internal.RuntimeState
	lrs, err := s.runtimeStatesDb.GetLatestWithReconcilerInputByRuntimeID(operation.RuntimeID)
	if err != nil {
		return s.operationManager.RetryOperation(operation, "error while getting latest runtime state", err, 5*time.Second, 1*time.Minute, log)
	}

	op, delay, _ := s.operationManager.UpdateOperation(operation, func(op *internal.UpdatingOperation) {
		op.LastRuntimeState = lrs
	}, log)
	log.Info("Last runtime state: ", op.LastRuntimeState.ID, ", service catalog migration triggered: ", operation.InstanceDetails.SCMigrationTriggered)
	return op, delay, nil
}
