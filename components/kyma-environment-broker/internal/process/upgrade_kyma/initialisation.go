package upgrade_kyma

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	orchestrationExt "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"

	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/sirupsen/logrus"
)

const (
	// the time after which the operation is marked as expired
	CheckStatusTimeout = 3 * time.Hour
)

type InitialisationStep struct {
	operationManager       *process.UpgradeKymaOperationManager
	operationStorage       storage.Operations
	instanceStorage        storage.Instances
	provisionerClient      provisioner.Client
	inputBuilder           input.CreatorForPlan
	timeSchedule           TimeSchedule
	runtimeVerConfigurator RuntimeVersionConfiguratorForUpgrade
}

func NewInitialisationStep(os storage.Operations, is storage.Instances, pc provisioner.Client, b input.CreatorForPlan, timeSchedule *TimeSchedule,
	rvc RuntimeVersionConfiguratorForUpgrade) *InitialisationStep {
	ts := timeSchedule
	if ts == nil {
		ts = &TimeSchedule{
			Retry:              5 * time.Second,
			StatusCheck:        time.Minute,
			UpgradeKymaTimeout: time.Hour,
		}
	}
	return &InitialisationStep{
		operationManager:       process.NewUpgradeKymaOperationManager(os),
		operationStorage:       os,
		instanceStorage:        is,
		provisionerClient:      pc,
		inputBuilder:           b,
		timeSchedule:           *ts,
		runtimeVerConfigurator: rvc,
	}
}

func (s *InitialisationStep) Name() string {
	return "Upgrade_Kyma_Initialisation"
}

func (s *InitialisationStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if operation.State == orchestrationExt.Canceled {
		log.Infof("Skipping processing because orchestration %s was canceled", operation.OrchestrationID)
		return s.operationManager.OperationCanceled(operation, fmt.Sprintf("orchestration %s was canceled", operation.OrchestrationID))
	}
	if operation.State == orchestrationExt.Pending {
		operation.State = orchestrationExt.InProgress

		op, err := s.operationStorage.UpdateUpgradeKymaOperation(operation)
		if err != nil {
			log.Errorf("while updating operation: %v", err)
			return operation, s.timeSchedule.Retry, nil
		}
		operation = *op
	}

	// rewrite necessary data from ProvisioningOperation to operation internal.UpgradeOperation
	provisioningOperation, err := s.operationStorage.GetProvisioningOperationByInstanceID(operation.InstanceID)
	if err != nil {
		log.Errorf("while getting provisioning operation from storage")
		return operation, s.timeSchedule.Retry, nil
	}
	if provisioningOperation.State == domain.InProgress {
		log.Info("waiting for provisioning operation to finish")
		return operation, s.timeSchedule.UpgradeKymaTimeout, nil
	}
	operation.ProvisioningParameters = provisioningOperation.ProvisioningParameters

	instance, err := s.instanceStorage.GetByID(operation.InstanceID)
	switch {
	case err == nil:
		if operation.ProvisionerOperationID == "" {
			// if schedule is maintenanceWindow and time window for this operation has finished we reprocess on next time window
			if !operation.MaintenanceWindowEnd.IsZero() && operation.MaintenanceWindowEnd.Before(time.Now()) {
				return s.rescheduleAtNextMaintenanceWindow(operation, log)
			}
			log.Info("provisioner operation ID is empty, initialize upgrade runtime input request")
			return s.initializeUpgradeRuntimeRequest(operation, log)
		}
		log.Infof("runtime being upgraded, check operation status")
		operation.InstanceDetails.RuntimeID = instance.RuntimeID
		return s.checkRuntimeStatus(operation, instance, log.WithField("runtimeID", instance.RuntimeID))
	case dberr.IsNotFound(err):
		log.Info("instance does not exist, it may have been deprovisioned")
		return s.operationManager.OperationSucceeded(operation, "instance was not found")
	default:
		log.Errorf("unable to get instance from storage: %s", err)
		return operation, s.timeSchedule.Retry, nil
	}

}

func (s *InitialisationStep) rescheduleAtNextMaintenanceWindow(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	operation.MaintenanceWindowBegin = operation.MaintenanceWindowBegin.AddDate(0, 0, 1)
	operation.MaintenanceWindowEnd = operation.MaintenanceWindowEnd.AddDate(0, 0, 1)
	operation, repeat := s.operationManager.UpdateOperation(operation)
	if repeat != 0 {
		log.Errorf("cannot save updated maintenance window to DB")
		return operation, s.timeSchedule.Retry, nil
	}
	until := time.Until(operation.MaintenanceWindowBegin)
	log.Infof("Upgrade operation %s will be rescheduled in %v", operation.Operation.ID, until)
	return operation, until, nil
}

func (s *InitialisationStep) initializeUpgradeRuntimeRequest(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if err := s.configureKymaVersion(&operation); err != nil {
		return s.operationManager.RetryOperation(operation, err.Error(), 5*time.Second, 5*time.Minute, log)
	}

	log.Infof("create provisioner input creator for plan ID %q", operation.ProvisioningParameters.PlanID)
	creator, err := s.inputBuilder.CreateUpgradeInput(operation.ProvisioningParameters, operation.RuntimeVersion)
	switch {
	case err == nil:
		operation.InputCreator = creator

		operation, repeat := s.operationManager.UpdateOperation(operation)
		if repeat != 0 {
			log.Errorf("cannot save the operation")
			return operation, time.Second, nil
		}

		return operation, 0, nil // go to next step
	case kebError.IsTemporaryError(err):
		log.Errorf("cannot create upgrade runtime input creator at the moment for plan %s: %s", operation.ProvisioningParameters.PlanID, err)
		return s.operationManager.RetryOperation(operation, err.Error(), 5*time.Second, 5*time.Minute, log)
	default:
		log.Errorf("cannot create input creator for plan %s: %s", operation.ProvisioningParameters.PlanID, err)
		return s.operationManager.OperationFailed(operation, "cannot create provisioning input creator")
	}
}

func (s *InitialisationStep) configureKymaVersion(operation *internal.UpgradeKymaOperation) error {
	if !operation.RuntimeVersion.IsEmpty() {
		return nil
	}
	version, err := s.runtimeVerConfigurator.ForUpgrade(*operation)
	if err != nil {
		return errors.Wrap(err, "while getting runtime version for upgrade")
	}
	operation.RuntimeVersion = *version

	var repeat time.Duration
	if *operation, repeat = s.operationManager.UpdateOperation(*operation); repeat != 0 {
		return errors.New("unable to update operation with RuntimeVersion property")
	}

	return nil
}

func (s *InitialisationStep) checkRuntimeStatus(operation internal.UpgradeKymaOperation, instance *internal.Instance, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > CheckStatusTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", CheckStatusTimeout))
	}

	status, err := s.provisionerClient.RuntimeOperationStatus(instance.GlobalAccountID, operation.ProvisionerOperationID)
	if err != nil {
		return operation, s.timeSchedule.StatusCheck, nil
	}
	log.Infof("call to provisioner returned %s status", status.State.String())

	var msg string
	if status.Message != nil {
		msg = *status.Message
	}

	switch status.State {
	case gqlschema.OperationStateSucceeded:
		return s.operationManager.OperationSucceeded(operation, msg)
	case gqlschema.OperationStateInProgress:
		return operation, s.timeSchedule.StatusCheck, nil
	case gqlschema.OperationStatePending:
		return operation, s.timeSchedule.StatusCheck, nil
	case gqlschema.OperationStateFailed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("provisioner client returns failed status: %s", msg))
	}

	return s.operationManager.OperationFailed(operation, fmt.Sprintf("unsupported provisioner client status: %s", status.State.String()))
}
