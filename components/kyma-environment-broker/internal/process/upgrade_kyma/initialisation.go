package upgrade_kyma

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	orchestrationExt "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"

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
	operationManager            *process.UpgradeKymaOperationManager
	operationStorage            storage.Operations
	orchestrationStorage        storage.Orchestrations
	instanceStorage             storage.Instances
	provisionerClient           provisioner.Client
	inputBuilder                input.CreatorForPlan
	evaluationManager           *avs.EvaluationManager
	timeSchedule                TimeSchedule
	runtimeVerConfigurator      RuntimeVersionConfiguratorForUpgrade
	serviceManagerClientFactory internal.SMClientFactory
}

func NewInitialisationStep(os storage.Operations, ors storage.Orchestrations, is storage.Instances, pc provisioner.Client, b input.CreatorForPlan, em *avs.EvaluationManager,
	timeSchedule *TimeSchedule, rvc RuntimeVersionConfiguratorForUpgrade, smcf internal.SMClientFactory) *InitialisationStep {
	ts := timeSchedule
	if ts == nil {
		ts = &TimeSchedule{
			Retry:              5 * time.Second,
			StatusCheck:        time.Minute,
			UpgradeKymaTimeout: time.Hour,
		}
	}
	return &InitialisationStep{
		operationManager:            process.NewUpgradeKymaOperationManager(os),
		operationStorage:            os,
		orchestrationStorage:        ors,
		instanceStorage:             is,
		provisionerClient:           pc,
		inputBuilder:                b,
		evaluationManager:           em,
		timeSchedule:                *ts,
		runtimeVerConfigurator:      rvc,
		serviceManagerClientFactory: smcf,
	}
}

func (s *InitialisationStep) Name() string {
	return "Upgrade_Kyma_Initialisation"
}

func (s *InitialisationStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	operation.SMClientFactory = s.serviceManagerClientFactory

	// Check concurrent deprovisioning (or suspension) operation (launched after target resolution)
	// Terminate (preempt) upgrade immediately with succeeded
	lastOp, err := s.operationStorage.GetLastOperation(operation.InstanceID)
	if err != nil {
		return operation, s.timeSchedule.Retry, nil
	}
	if lastOp.Type == internal.OperationTypeDeprovision {
		return s.operationManager.OperationSucceeded(operation, fmt.Sprintf("operation preempted by deprovisioning %s", lastOp.ID), log)
	}

	if operation.State == orchestrationExt.Pending {
		// Check if the orchestration got cancelled, don't start new pending operation
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

		// rewrite necessary data from ProvisioningOperation to operation internal.UpgradeOperation
		provisioningOperation, err := s.operationStorage.GetProvisioningOperationByInstanceID(operation.InstanceID)
		if err != nil {
			log.Errorf("while getting provisioning operation from storage")
			return operation, s.timeSchedule.Retry, nil
		}
		op, delay, _ := s.operationManager.UpdateOperation(operation, func(op *internal.UpgradeKymaOperation) {
			op.ProvisioningParameters = provisioningOperation.ProvisioningParameters
			if op.ProvisioningParameters.ErsContext.SMOperatorCredentials == nil && lastOp.ProvisioningParameters.ErsContext.SMOperatorCredentials != nil {
				op.ProvisioningParameters.ErsContext.SMOperatorCredentials = lastOp.ProvisioningParameters.ErsContext.SMOperatorCredentials
			}
			op.State = domain.InProgress
		}, log)
		if delay != 0 {
			return operation, delay, nil
		}
		operation = op
	}

	if operation.ProvisionerOperationID == "" {
		log.Info("provisioner operation ID is empty, initialize upgrade runtime input request")
		return s.initializeUpgradeRuntimeRequest(operation, log)
	}

	log.Infof("runtime being upgraded, check operation status")
	return s.checkRuntimeStatus(operation, log.WithField("runtimeID", operation.RuntimeOperation.RuntimeID))

}

func (s *InitialisationStep) initializeUpgradeRuntimeRequest(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if err := s.configureKymaVersion(&operation, log); err != nil {
		return s.operationManager.RetryOperation(operation, "while configuring kyma version", err, 5*time.Second, 5*time.Minute, log)
	}

	log.Infof("create provisioner input creator for plan ID %q", operation.ProvisioningParameters.PlanID)
	creator, err := s.inputBuilder.CreateUpgradeInput(operation.ProvisioningParameters, operation.RuntimeVersion)
	switch {
	case err == nil:
		operation.InputCreator = creator
		return operation, 0, nil // go to next step
	case kebError.IsTemporaryError(err):
		log.Errorf("cannot create upgrade runtime input creator at the moment for plan %s: %s", operation.ProvisioningParameters.PlanID, err)
		return s.operationManager.RetryOperation(operation, "while creating runtime input creator", err, 5*time.Second, 5*time.Minute, log)
	default:
		log.Errorf("cannot create input creator for plan %s: %s", operation.ProvisioningParameters.PlanID, err)
		return s.operationManager.OperationFailed(operation, "cannot create provisioning input creator", err, log)
	}
}

func (s *InitialisationStep) configureKymaVersion(operation *internal.UpgradeKymaOperation, log logrus.FieldLogger) error {
	if !operation.RuntimeVersion.IsEmpty() {
		return nil
	}

	// set Kyma version from request or runtime parameters
	var (
		err     error
		version *internal.RuntimeVersionData
	)

	version, err = s.runtimeVerConfigurator.ForUpgrade(*operation)
	if err != nil {
		return errors.Wrap(err, "while getting runtime version for upgrade")
	}

	// update operation version
	var repeat time.Duration
	if *operation, repeat, err = s.operationManager.UpdateOperation(*operation, func(operation *internal.UpgradeKymaOperation) {
		operation.RuntimeVersion = *version
	}, log); repeat != 0 {
		return errors.Wrap(err, "unable to update operation with RuntimeVersion property")
	}

	return nil
}

// checkRuntimeStatus will check operation runtime status
// It will also trigger performRuntimeTasks upgrade steps to ensure
// all the required dependencies have been fulfilled for upgrade operation.
func (s *InitialisationStep) checkRuntimeStatus(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > CheckStatusTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", CheckStatusTimeout), errors.New(""), log)
	}

	// Ensure AVS evaluations are set to maintenance
	operation, err := SetAvsStatusMaintenance(s.evaluationManager, s.operationManager, operation, log)
	if err != nil {
		if kebError.IsTemporaryError(err) {
			return s.operationManager.RetryOperation(operation, err.Error(), errors.New(""), 10*time.Second, 10*time.Minute, log)
		}
		return s.operationManager.OperationFailed(operation, err.Error(), errors.New(""), log)
	}

	if operation.ClusterConfigurationVersion != 0 {
		// upgrade was trigerred in reconciler, no need to call provisioner and create UpgradeRuntimeInput
		// TODO: deal with skipping steps in case of calling reconciler for Kyma 2.0 upgrade
		log.Debugf("Cluster configuration already created, skipping")
		return operation, 0, nil
	}

	status, err := s.provisionerClient.RuntimeOperationStatus(operation.RuntimeOperation.GlobalAccountID, operation.ProvisionerOperationID)
	if err != nil {
		return operation, s.timeSchedule.StatusCheck, nil
	}
	log.Infof("call to provisioner returned %s status", status.State.String())

	var msg string
	var delay time.Duration
	if status.Message != nil {
		msg = *status.Message
	}

	// wait for operation completion
	switch status.State {
	case gqlschema.OperationStateInProgress, gqlschema.OperationStatePending:
		return operation, s.timeSchedule.StatusCheck, nil
	case gqlschema.OperationStateSucceeded, gqlschema.OperationStateFailed:
		// Set post-upgrade description which also reset UpdatedAt for operation retries to work properly
		if operation.Description != postUpgradeDescription {
			operation, delay, _ = s.operationManager.UpdateOperation(operation, func(operation *internal.UpgradeKymaOperation) {
				operation.Description = postUpgradeDescription
			}, log)
			if delay != 0 {
				return operation, delay, nil
			}
		}
	}

	// Kyma 1.X operation is finished or failed, restore AVS status
	operation, err = RestoreAvsStatus(s.evaluationManager, s.operationManager, operation, log)
	if err != nil {
		if kebError.IsTemporaryError(err) {
			return s.operationManager.RetryOperation(operation, err.Error(), errors.New(""), 10*time.Second, 10*time.Minute, log)
		}
		return s.operationManager.OperationFailed(operation, err.Error(), errors.New(""), log)
	}

	// handle operation completion
	switch status.State {
	case gqlschema.OperationStateSucceeded:
		return s.operationManager.OperationSucceeded(operation, msg, log)
	case gqlschema.OperationStateFailed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("provisioner client returns failed status: %s", msg), errors.New(""), log)
	}

	return s.operationManager.OperationFailed(operation, fmt.Sprintf("unsupported provisioner client status: %s", status.State.String()), errors.New(""), log)
}
