package update

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtimeversion"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type InitKymaVersionStep struct {
	operationManager       *process.UpdateOperationManager
	runtimeVerConfigurator *runtimeversion.RuntimeVersionConfigurator
	runtimeStatesDb        storage.RuntimeStates
}

func NewInitKymaVersionStep(os storage.Operations, rvc *runtimeversion.RuntimeVersionConfigurator, runtimeStatesDb storage.RuntimeStates) *InitKymaVersionStep {
	return &InitKymaVersionStep{
		operationManager:       process.NewUpdateOperationManager(os),
		runtimeVerConfigurator: rvc,
		runtimeStatesDb:        runtimeStatesDb,
	}
}

func (s *InitKymaVersionStep) Name() string {
	return "Update_Init_Kyma_Version"
}

func (s *InitKymaVersionStep) Run(operation internal.UpdatingOperation, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	var version *internal.RuntimeVersionData
	var err error
	if operation.RuntimeVersion.IsEmpty() {
		version, err = s.runtimeVerConfigurator.ForUpdating(operation)
		if err != nil {
			return s.operationManager.RetryOperation(operation, err.Error(), 5*time.Second, 1*time.Minute, log)
		}
	}
	var lrs internal.RuntimeState
	if version.MajorVersion == 2 {
		lrs, err = s.runtimeStatesDb.GetLatestWithReconcilerInputByRuntimeID(operation.RuntimeID)
		if err != nil {
			return s.operationManager.RetryOperation(operation, err.Error(), 5*time.Second, 1*time.Minute, log)
		}
	}
	op, delay := s.operationManager.UpdateOperation(operation, func(op *internal.UpdatingOperation) {
		if version != nil {
			op.RuntimeVersion = *version
		}
		op.LastRuntimeState = lrs
	}, log)
	log.Info("Init runtime version: ", op.RuntimeVersion.MajorVersion, ", last runtime state: ", op.LastRuntimeState.ID, ", service catalog migration triggered: ", operation.InstanceDetails.SCMigrationTriggered)
	return op, delay, nil
}
