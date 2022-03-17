package upgrade_kyma

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/sirupsen/logrus"
)

func SetAvsStatusMaintenance(evaluationManager *avs.EvaluationManager, operationManager *process.UpgradeKymaOperationManager, operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, error) {
	hasMonitors := evaluationManager.HasMonitors(operation.Avs)
	inMaintenance := evaluationManager.InMaintenance(operation.Avs)
	var err error = nil
	var delay time.Duration = 0

	if hasMonitors && !inMaintenance {
		log.Infof("setting AVS evaluations statuses to maintenance")
		err = evaluationManager.SetMaintenanceStatus(&operation.Avs, log)
		operation, delay, _ = operationManager.UpdateOperation(operation, func(op *internal.UpgradeKymaOperation) {
			op.Avs.AvsInternalEvaluationStatus = operation.Avs.AvsInternalEvaluationStatus
			op.Avs.AvsExternalEvaluationStatus = operation.Avs.AvsExternalEvaluationStatus
		}, log)
		if err == nil && delay > 0 {
			err = kebError.NewTemporaryError("failed to update avs status in operation")
		}
	}

	return operation, err
}

func RestoreAvsStatus(evaluationManager *avs.EvaluationManager, operationManager *process.UpgradeKymaOperationManager, operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, error) {
	hasMonitors := evaluationManager.HasMonitors(operation.Avs)
	inMaintenance := evaluationManager.InMaintenance(operation.Avs)
	var err error = nil
	var delay time.Duration = 0

	if hasMonitors && inMaintenance {
		log.Infof("clearing AVS maintenantce statuses and restoring original AVS evaluation statuses")
		err = evaluationManager.RestoreStatus(&operation.Avs, log)
		operation, delay, _ = operationManager.UpdateOperation(operation, func(op *internal.UpgradeKymaOperation) {
			op.Avs.AvsInternalEvaluationStatus = operation.Avs.AvsInternalEvaluationStatus
			op.Avs.AvsExternalEvaluationStatus = operation.Avs.AvsExternalEvaluationStatus
		}, log)
		if err == nil && delay > 0 {
			err = kebError.NewTemporaryError("failed to update avs status in operation")
		}
	}

	return operation, err
}
