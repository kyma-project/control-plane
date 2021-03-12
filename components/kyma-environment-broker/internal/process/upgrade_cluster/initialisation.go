package upgrade_cluster

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/sirupsen/logrus"
)

const (
	UpgradeInitSteps int = iota + 1
	UpgradeFinishSteps
)

const (
	// the time after which the operation is marked as expired
	CheckStatusTimeout = 3 * time.Hour
)

const postUpgradeDescription = "Performing post-upgrade tasks"

type InitialisationStep struct {
	operationManager     *process.UpgradeClusterOperationManager
	operationStorage     storage.Operations
	orchestrationStorage storage.Orchestrations
	provisionerClient    provisioner.Client
	inputBuilder         input.CreatorForPlan
	evaluationManager    *avs.EvaluationManager
	timeSchedule         TimeSchedule
}

func NewInitialisationStep(os storage.Operations, ors storage.Orchestrations, pc provisioner.Client, b input.CreatorForPlan, em *avs.EvaluationManager,
	timeSchedule *TimeSchedule) *InitialisationStep {
	ts := timeSchedule
	if ts == nil {
		ts = &TimeSchedule{
			Retry:                 5 * time.Second,
			StatusCheck:           time.Minute,
			UpgradeClusterTimeout: time.Hour,
		}
	}
	return &InitialisationStep{
		operationManager:     process.NewUpgradeClusterOperationManager(os),
		operationStorage:     os,
		orchestrationStorage: ors,
		provisionerClient:    pc,
		inputBuilder:         b,
		evaluationManager:    em,
		timeSchedule:         *ts,
	}
}

func (s *InitialisationStep) Name() string {
	return "Upgrade_Kyma_Initialisation"
}

func (s *InitialisationStep) Run(operation internal.UpgradeClusterOperation, log logrus.FieldLogger) (internal.UpgradeClusterOperation, time.Duration, error) {
	// Check concurrent deprovisioning (or suspension) operation (launched after target resolution)
	// Terminate (preempt) upgrade immediately with succeeded
	lastOp, err := s.operationStorage.GetLastOperation(operation.InstanceID)
	if err != nil {
		return operation, s.timeSchedule.Retry, nil
	}
	if lastOp.Type == internal.OperationTypeDeprovision {
		return s.operationManager.OperationSucceeded(operation, fmt.Sprintf("operation preempted by deprovisioning %s", lastOp.ID))
	}

	if operation.State == orchestration.Pending {
		// Check if the orchestreation got cancelled, don't start new pending operation
		orchestration, err := s.orchestrationStorage.GetByID(operation.OrchestrationID)
		if err != nil {
			return operation, s.timeSchedule.Retry, nil
		}
		if orchestration.IsCanceled() {
			log.Infof("Skipping processing because orchestration %s was canceled", operation.OrchestrationID)
			return s.operationManager.OperationCanceled(operation, fmt.Sprintf("orchestration %s was canceled", operation.OrchestrationID))
		}

		// Check concurrent operations and wait to finish before proceeding
		// - unsuspension provisioning launched after suspension
		// - kyma upgrade or cluster upgrade
		switch lastOp.Type {
		case internal.OperationTypeProvision, internal.OperationTypeUpgradeKyma, internal.OperationTypeUpgradeCluster:
			if !lastOp.IsFinished() {
				return operation, s.timeSchedule.StatusCheck, nil
			}
		}

		operation.State = domain.InProgress

		op, err := s.operationStorage.UpdateUpgradeClusterOperation(operation)
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
	operation.ProvisioningParameters = provisioningOperation.ProvisioningParameters

	if operation.ProvisionerOperationID == "" {
		log.Info("provisioner operation ID is empty, initialize upgrade shoot input request")
		return s.initializeUpgradeShootRequest(operation, log)
	}

	log.Infof("runtime being upgraded, check operation status")
	return s.checkRuntimeStatus(operation, log.WithField("runtimeID", operation.RuntimeOperation.RuntimeID))
}

func (s *InitialisationStep) initializeUpgradeShootRequest(operation internal.UpgradeClusterOperation, log logrus.FieldLogger) (internal.UpgradeClusterOperation, time.Duration, error) {
	log.Infof("create provisioner input creator for plan ID %q", operation.ProvisioningParameters.PlanID)
	creator, err := s.inputBuilder.CreateUpgradeShootInput(operation.ProvisioningParameters)
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
		log.Errorf("cannot create upgrade shoot input creator at the moment for plan %s: %s", operation.ProvisioningParameters.PlanID, err)
		return s.operationManager.RetryOperation(operation, err.Error(), 5*time.Second, 5*time.Minute, log)
	default:
		log.Errorf("cannot create input creator for plan %s: %s", operation.ProvisioningParameters.PlanID, err)
		return s.operationManager.OperationFailed(operation, "cannot create provisioning input creator")
	}
}

// performRuntimeTasks Ensures that required logic on init and finish is executed.
// Uses internal and external Avs monitor statuses to verify state.
func (s *InitialisationStep) performRuntimeTasks(step int, operation internal.UpgradeClusterOperation, log logrus.FieldLogger) (internal.UpgradeClusterOperation, time.Duration, error) {
	hasMonitors := s.evaluationManager.HasMonitors(operation.Avs)
	inMaintenance := s.evaluationManager.InMaintenance(operation.Avs)
	var err error = nil
	var delay time.Duration = 0

	switch step {
	case UpgradeInitSteps:
		if hasMonitors && !inMaintenance {
			log.Infof("executing init upgrade steps")
			err = s.evaluationManager.SetMaintenanceStatus(&operation.Avs, log)
			operation, delay = s.operationManager.UpdateOperation(operation)
		}
	case UpgradeFinishSteps:
		if hasMonitors && inMaintenance {
			log.Infof("executing finish upgrade steps")
			err = s.evaluationManager.RestoreStatus(&operation.Avs, log)
			operation, delay = s.operationManager.UpdateOperation(operation)
		}
	}

	switch {
	case err == nil:
		return operation, delay, nil
	case kebError.IsTemporaryError(err):
		return s.operationManager.RetryOperation(operation, err.Error(), 10*time.Second, 10*time.Minute, log)
	default:
		return s.operationManager.OperationFailed(operation, err.Error())
	}
}

// checkRuntimeStatus will check operation runtime status
// It will also trigger performRuntimeTasks upgrade steps to ensure
// all the required dependencies have been fulfilled for upgrade operation.
func (s *InitialisationStep) checkRuntimeStatus(operation internal.UpgradeClusterOperation, log logrus.FieldLogger) (internal.UpgradeClusterOperation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > CheckStatusTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", CheckStatusTimeout))
	}

	status, err := s.provisionerClient.RuntimeOperationStatus(operation.RuntimeOperation.GlobalAccountID, operation.ProvisionerOperationID)
	if err != nil {
		return operation, s.timeSchedule.StatusCheck, nil
	}
	log.Infof("call to provisioner returned %s status", status.State.String())

	var msg string
	if status.Message != nil {
		msg = *status.Message
	}

	// do required steps on init
	operation, delay, err := s.performRuntimeTasks(UpgradeInitSteps, operation, log)
	if delay != 0 || err != nil {
		return operation, delay, err
	}

	// wait for operation completion
	switch status.State {
	case gqlschema.OperationStateInProgress, gqlschema.OperationStatePending:
		return operation, s.timeSchedule.StatusCheck, nil
	case gqlschema.OperationStateSucceeded, gqlschema.OperationStateFailed:
		// Set post-upgrade description which also reset UpdatedAt for operation retries to work properly
		if operation.Description != postUpgradeDescription {
			operation.Description = postUpgradeDescription
			operation, delay = s.operationManager.UpdateOperation(operation)
			if delay != 0 {
				return operation, delay, nil
			}
		}
	}

	// do required steps on finish
	operation, delay, err = s.performRuntimeTasks(UpgradeFinishSteps, operation, log)
	if delay != 0 || err != nil {
		return operation, delay, err
	}

	// handle operation completion
	switch status.State {
	case gqlschema.OperationStateSucceeded:
		return s.operationManager.OperationSucceeded(operation, msg)
	case gqlschema.OperationStateFailed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("provisioner client returns failed status: %s", msg))
	}

	return s.operationManager.OperationFailed(operation, fmt.Sprintf("unsupported provisioner client status: %s", status.State.String()))
}
