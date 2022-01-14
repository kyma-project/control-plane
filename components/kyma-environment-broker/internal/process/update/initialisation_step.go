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
	operationManager *process.UpdateOperationManager
	operationStorage storage.Operations
	instanceStorage  storage.Instances
	inputBuilder     input.CreatorForPlan
}

func NewInitialisationStep(is storage.Instances, os storage.Operations, b input.CreatorForPlan) *InitialisationStep {
	return &InitialisationStep{
		operationManager: process.NewUpdateOperationManager(os),
		operationStorage: os,
		instanceStorage:  is,
		inputBuilder:     b,
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

	if operation.State == orchestration.Pending {
		if !lastOp.IsFinished() {
			return operation, time.Minute, nil
		}

		// read the instsance details (it could happen that created updating operation has outdated one)
		instance, err := s.instanceStorage.GetByID(operation.InstanceID)
		if err != nil {
			return operation, time.Second, nil
		}
		if operation.ProvisioningParameters.ErsContext.SMOperatorCredentials != nil {
			instance.Parameters.ErsContext.SMOperatorCredentials = operation.ProvisioningParameters.ErsContext.SMOperatorCredentials
		}
		if _, err := s.instanceStorage.Update(*instance); err != nil {
			return operation, time.Second, err
		}

		op, delay, _ := s.operationManager.UpdateOperation(operation, func(op *internal.UpdatingOperation) {
			op.State = domain.InProgress
			op.InstanceDetails = instance.InstanceDetails
			op.InstanceDetails.SCMigrationTriggered = op.ProvisioningParameters.ErsContext.IsMigration
			if op.ProvisioningParameters.ErsContext.SMOperatorCredentials == nil && lastOp.ProvisioningParameters.ErsContext.SMOperatorCredentials != nil {
				op.ProvisioningParameters.ErsContext.SMOperatorCredentials = lastOp.ProvisioningParameters.ErsContext.SMOperatorCredentials
			}
		}, log)
		if delay != 0 {
			return operation, delay, nil
		}
		operation = op
	}

	if lastOp.Type == internal.OperationTypeDeprovision {
		return s.operationManager.OperationSucceeded(operation, fmt.Sprintf("operation preempted by deprovisioning %s", lastOp.ID), log)
	}

	return s.initializeUpgradeShootRequest(operation, log)
}

func (s *InitialisationStep) initializeUpgradeShootRequest(operation internal.UpdatingOperation, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
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
		return s.operationManager.RetryOperation(operation, err.Error(), err, 5*time.Second, 1*time.Minute, log)
	default:
		log.Errorf("cannot create input creator for plan %s: %s", operation.ProvisioningParameters.PlanID, err)
		return s.operationManager.OperationFailed(operation, "cannot create provisioning input creator", err, log)
	}
}
