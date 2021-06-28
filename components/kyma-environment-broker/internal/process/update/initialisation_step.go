package update

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
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
	operationManager     *process.UpdateOperationManager
	operationStorage     storage.Operations
	inputBuilder         input.CreatorForPlan
}

func NewInitialisationStep(os storage.Operations, b input.CreatorForPlan) *InitialisationStep {
	return &InitialisationStep{
		operationManager:     process.NewUpdateOperationManager(os),
		operationStorage:     os,
		inputBuilder:         b,
	}
}

func (s *InitialisationStep) Name() string {
	return "Upgrade_Kyma_Initialisation"
}

func (s *InitialisationStep) Run(operation internal.UpdatingOperation, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	// Check concurrent deprovisioning (or suspension) operation (launched after target resolution)
	// Terminate (preempt) upgrade immediately with succeeded
	lastOp, err := s.operationStorage.GetLastOperation(operation.InstanceID)
	if err != nil {
		return operation, time.Minute, nil
	}
	if lastOp.Type == internal.OperationTypeDeprovision {
		return s.operationManager.OperationSucceeded(operation, fmt.Sprintf("operation preempted by deprovisioning %s", lastOp.ID), log)
	}

	if operation.State == orchestration.Pending {


		// Check concurrent operations and wait to finish before proceeding
		// - unsuspension provisioning launched after suspension
		// - kyma upgrade or cluster upgrade
		switch lastOp.Type {
		case internal.OperationTypeProvision, internal.OperationTypeUpgradeKyma, internal.OperationTypeUpgradeCluster:
			if !lastOp.IsFinished() {
				return operation, time.Minute, nil
			}
		}

		op, delay := s.operationManager.UpdateOperation(operation, func(op *internal.UpdatingOperation) {
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

	log.Infof("runtime being upgraded, check operation status")
	return operation, 0, nil
}

func (s *InitialisationStep) initializeUpgradeShootRequest(operation internal.UpdatingOperation, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	// rewrite necessary data from ProvisioningOperation to operation internal.UpgradeOperation
	//provisioningOperation, err := s.operationStorage.GetProvisioningOperationByInstanceID(operation.InstanceID)
	//if err != nil {
	//	log.Errorf("while getting provisioning operation from storage")
	//	return operation, time.Second, nil
	//}
	//
	//operation, delay := s.operationManager.UpdateOperation(operation, func(op *internal.UpdatingOperation) {
	//	op.ProvisioningParameters = provisioningOperation.ProvisioningParameters
	//	op.
	//}, log)
	//if delay != 0 {
	//	return operation, delay, nil
	//}

	pp := operation.ProvisioningParameters
	pp.Parameters.OIDC = operation.UpdatingParameters.OIDC
	log.Infof("create provisioner input creator for plan ID %q", operation.ProvisioningParameters)
	creator, err := s.inputBuilder.CreateUpgradeShootInput(operation.ProvisioningParameters)
	switch {
	case err == nil:
		operation.InputCreator = creator
		return operation, 0, nil // go to next step
	case kebError.IsTemporaryError(err):
		log.Errorf("cannot create upgrade shoot input creator at the moment for plan %s: %s", operation.ProvisioningParameters.PlanID, err)
		return s.operationManager.RetryOperation(operation, err.Error(), 5*time.Second, 5*time.Minute, log)
	default:
		log.Errorf("cannot create input creator for plan %s: %s", operation.ProvisioningParameters.PlanID, err)
		return s.operationManager.OperationFailed(operation, "cannot create provisioning input creator", log)
	}
}

