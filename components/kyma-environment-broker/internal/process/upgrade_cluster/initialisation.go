package upgrade_cluster

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/notification"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v8/domain"
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
	bundleBuilder        notification.BundleBuilder
}

func NewInitialisationStep(os storage.Operations, ors storage.Orchestrations, pc provisioner.Client, b input.CreatorForPlan, em *avs.EvaluationManager,
	timeSchedule *TimeSchedule, bundleBuilder notification.BundleBuilder) *InitialisationStep {
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
		bundleBuilder:        bundleBuilder,
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
		return s.operationManager.OperationSucceeded(operation, fmt.Sprintf("operation preempted by deprovisioning %s", lastOp.ID), log)
	}

	if operation.State == orchestration.Pending {
		// Check if the orchestreation got cancelled, don't start new pending operation
		orchestration, err := s.orchestrationStorage.GetByID(operation.OrchestrationID)
		if err != nil {
			return operation, s.timeSchedule.Retry, nil
		}
		if orchestration.IsCanceled() {
			log.Infof("Skipping processing because orchestration %s was canceled", operation.OrchestrationID)
			return s.operationManager.OperationCanceled(operation, fmt.Sprintf("orchestration %s was canceled", operation.OrchestrationID), log)
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

		op, delay, _ := s.operationManager.UpdateOperation(operation, func(op *internal.UpgradeClusterOperation) {
			op.ProvisioningParameters.ErsContext = internal.UpdateERSContext(op.ProvisioningParameters.ErsContext, lastOp.ProvisioningParameters.ErsContext)
			op.State = domain.InProgress
		}, log)
		if delay != 0 {
			return operation, delay, nil
		}
		operation = op
	}

	if operation.ProvisionerOperationID == "" {
		log.Info("provisioner operation ID is empty, initialize upgrade shoot input request")
		return s.initializeUpgradeShootRequest(operation, log)
	}

	log.Infof("runtime being upgraded, check operation status for provisioner operation id: %v", operation.ProvisionerOperationID)
	return s.checkRuntimeStatus(operation, log.WithField("runtimeID", operation.RuntimeOperation.RuntimeID))
}

func (s *InitialisationStep) initializeUpgradeShootRequest(operation internal.UpgradeClusterOperation, log logrus.FieldLogger) (internal.UpgradeClusterOperation, time.Duration, error) {
	log.Infof("create provisioner input creator for plan ID %q", operation.ProvisioningParameters)
	creator, err := s.inputBuilder.CreateUpgradeShootInput(operation.ProvisioningParameters)
	switch {
	case err == nil:
		operation.InputCreator = creator
		return operation, 0, nil // go to next step
	case kebError.IsTemporaryError(err):
		log.Errorf("cannot create upgrade shoot input creator at the moment for plan %s: %s", operation.ProvisioningParameters.PlanID, err)
		return s.operationManager.RetryOperation(operation, "error while creating upgrade shoot input creator", err, 5*time.Second, 5*time.Minute, log)
	default:
		log.Errorf("cannot create input creator for plan %s: %s", operation.ProvisioningParameters.PlanID, err)
		return s.operationManager.OperationFailed(operation, "cannot create provisioning input creator", err, log)
	}
}

// performRuntimeTasks Ensures that required logic on init and finish is executed.
// Uses internal and external Avs monitor statuses to verify state.
func (s *InitialisationStep) performRuntimeTasks(step int, operation internal.UpgradeClusterOperation, log logrus.FieldLogger) (internal.UpgradeClusterOperation, time.Duration, error) {
	hasMonitors := s.evaluationManager.HasMonitors(operation.Avs)
	inMaintenance := s.evaluationManager.InMaintenance(operation.Avs)
	var err error = nil
	var delay time.Duration = 0
	var updateAvsStatus = func(op *internal.UpgradeClusterOperation) {
		op.Avs.AvsInternalEvaluationStatus = operation.Avs.AvsInternalEvaluationStatus
		op.Avs.AvsExternalEvaluationStatus = operation.Avs.AvsExternalEvaluationStatus
	}

	switch step {
	case UpgradeInitSteps:
		if hasMonitors && !inMaintenance {
			log.Infof("executing init upgrade steps")
			err = s.evaluationManager.SetMaintenanceStatus(&operation.Avs, log)
			operation, delay, _ = s.operationManager.UpdateOperation(operation, updateAvsStatus, log)
		}
	case UpgradeFinishSteps:
		if hasMonitors && inMaintenance {
			log.Infof("executing finish upgrade steps")
			err = s.evaluationManager.RestoreStatus(&operation.Avs, log)
			operation, delay, _ = s.operationManager.UpdateOperation(operation, updateAvsStatus, log)
		}
	}

	switch {
	case err == nil:
		return operation, delay, nil
	case kebError.IsTemporaryError(err):
		return s.operationManager.RetryOperation(operation, "error while performing runtime tasks", err, 10*time.Second, 10*time.Minute, log)
	default:
		return s.operationManager.OperationFailed(operation, "error while performing runtime tasks", err, log)
	}
}

// checkRuntimeStatus will check operation runtime status
// It will also trigger performRuntimeTasks upgrade steps to ensure
// all the required dependencies have been fulfilled for upgrade operation.
func (s *InitialisationStep) checkRuntimeStatus(operation internal.UpgradeClusterOperation, log logrus.FieldLogger) (internal.UpgradeClusterOperation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > CheckStatusTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		//send cunstomer notification
		if !s.bundleBuilder.DisabledCheck() {
			err := s.sendNotificationComplete(operation, log)
			//currently notification error can only be temporary error
			if err != nil && kebError.IsTemporaryError(err) {
				return operation, 5 * time.Second, nil
			}
		}
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", CheckStatusTimeout), nil, log)
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
		//send cunstomer notification
		if !s.bundleBuilder.DisabledCheck() {
			err := s.sendNotificationComplete(operation, log)
			//currently notification error can only be temporary error
			if err != nil && kebError.IsTemporaryError(err) {
				return operation, 5 * time.Second, nil
			}
		}
		// Set post-upgrade description which also reset UpdatedAt for operation retries to work properly
		if operation.Description != postUpgradeDescription {
			operation, delay, _ = s.operationManager.UpdateOperation(operation, func(operation *internal.UpgradeClusterOperation) {
				operation.Description = postUpgradeDescription
			}, log)
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
		return s.operationManager.OperationSucceeded(operation, msg, log)
	case gqlschema.OperationStateFailed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("provisioner client returns failed status: %s", msg), nil, log)
	}

	return s.operationManager.OperationFailed(operation, fmt.Sprintf("unsupported provisioner client status: %s", status.State.String()), nil, log)
}

func (s *InitialisationStep) sendNotificationComplete(operation internal.UpgradeClusterOperation, log logrus.FieldLogger) error {
	tenants := []notification.NotificationTenant{
		{
			InstanceID: operation.InstanceID,
			EndDate:    time.Now().Format("2006-01-02 15:04:05"),
			State:      notification.FinishedMaintenanceState,
		},
	}
	notificationParams := notification.NotificationParams{
		OrchestrationID: operation.OrchestrationID,
		Tenants:         tenants,
	}
	notificationBundle, err := s.bundleBuilder.NewBundle(operation.OrchestrationID, notificationParams)
	if err != nil {
		log.Errorf("%s: %s", "Failed to create Notification Bundle", err)
		return err
	}
	err2 := notificationBundle.UpdateNotificationEvent()
	if err2 != nil {
		msg := fmt.Sprintf("cannot update notification for orchestration %s", operation.OrchestrationID)
		log.Errorf("%s: %s", msg, err)
		return err
	}
	return nil
}
