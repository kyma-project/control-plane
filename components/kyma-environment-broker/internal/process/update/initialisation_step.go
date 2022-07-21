package update

import (
	"errors"
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"

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

//go:generate mockery --name=RuntimeVersionConfiguratorForUpdating --output=automock --outpkg=automock --case=underscore
type RuntimeVersionConfiguratorForUpdating interface {
	ForUpdating(op internal.UpdatingOperation) (*internal.RuntimeVersionData, error)
}

type InitialisationStep struct {
	operationManager       *process.UpdateOperationManager
	operationStorage       storage.Operations
	instanceStorage        storage.Instances
	runtimeStatesStorage   storage.RuntimeStates
	runtimeVerConfigurator RuntimeVersionConfiguratorForUpdating
	inputBuilder           input.CreatorForPlan
}

func NewInitialisationStep(is storage.Instances, os storage.Operations, rs storage.RuntimeStates,
	rvc RuntimeVersionConfiguratorForUpdating, b input.CreatorForPlan) *InitialisationStep {
	return &InitialisationStep{
		operationManager:       process.NewUpdateOperationManager(os),
		operationStorage:       os,
		instanceStorage:        is,
		runtimeStatesStorage:   rs,
		runtimeVerConfigurator: rvc,
		inputBuilder:           b,
	}
}

func (s *InitialisationStep) Name() string {
	return "Update_Kyma_Initialisation"
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
			log.Infof("waiting for %s operation (%s) to be finished", lastOp.Type, lastOp.ID)
			return operation, time.Minute, nil
		}

		// read the instsance details (it could happen that created updating operation has outdated one)
		instance, err := s.instanceStorage.GetByID(operation.InstanceID)
		if err != nil {
			if dberr.IsNotFound(err) {
				log.Warnf("the instance already deprovisioned")
				return s.operationManager.OperationFailed(operation, "the instance was already deprovisioned", err, log)
			}
			return operation, time.Second, nil
		}
		instance.Parameters.ErsContext = internal.UpdateERSContext(instance.Parameters.ErsContext, operation.ProvisioningParameters.ErsContext)
		if _, err := s.instanceStorage.Update(*instance); err != nil {
			log.Errorf("unable to update the instance, retrying")
			return operation, time.Second, err
		}

		// suspension cleared runtimeID
		if operation.RuntimeID == "" {
			err = s.getRuntimeIdFromProvisioningOp(&operation, log)
			if err != nil {
				return s.operationManager.RetryOperation(operation, "error while getting runtime version", err, 5*time.Second, 1*time.Minute, log)
			}
		}

		version, err := s.runtimeVerConfigurator.ForUpdating(operation)
		if err != nil {
			return s.operationManager.RetryOperation(operation, "error while getting runtime version", err, 5*time.Second, 1*time.Minute, log)
		}

		op, delay, _ := s.operationManager.UpdateOperation(operation, func(op *internal.UpdatingOperation) {
			op.State = domain.InProgress
			op.InstanceDetails = instance.InstanceDetails
			op.InstanceDetails.SCMigrationTriggered = op.ProvisioningParameters.ErsContext.IsMigration
			if op.ProvisioningParameters.ErsContext.SMOperatorCredentials == nil && lastOp.ProvisioningParameters.ErsContext.SMOperatorCredentials != nil {
				op.ProvisioningParameters.ErsContext.SMOperatorCredentials = lastOp.ProvisioningParameters.ErsContext.SMOperatorCredentials
			}
			op.ProvisioningParameters.ErsContext = internal.UpdateERSContext(op.ProvisioningParameters.ErsContext, lastOp.ProvisioningParameters.ErsContext)
			if version != nil {
				op.RuntimeVersion = *version
			}
		}, log)
		if delay != 0 {
			log.Errorf("unable to update the operation (move to 'in progress'), retrying")
			return operation, delay, nil
		}
		operation = op
	}

	if lastOp.Type == internal.OperationTypeDeprovision {
		return s.operationManager.OperationSucceeded(operation, fmt.Sprintf("operation preempted by deprovisioning %s", lastOp.ID), log)
	}

	return s.initializeUpgradeShootRequest(operation, log)
}

func (s *InitialisationStep) getRuntimeIdFromProvisioningOp(operation *internal.UpdatingOperation, log logrus.FieldLogger) error {
	provOp, err := s.operationStorage.GetProvisioningOperationByInstanceID(operation.InstanceID)
	if err != nil {
		return errors.New("cannot get last provisioning operation for runtime id")
	}
	operation.RuntimeID = provOp.RuntimeID
	return nil
}

func (s *InitialisationStep) initializeUpgradeShootRequest(operation internal.UpdatingOperation, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	log.Infof("create provisioner input creator for plan ID %q", operation.ProvisioningParameters)
	creator, err := s.inputBuilder.CreateUpgradeShootInput(operation.ProvisioningParameters, operation.RuntimeVersion)
	switch {
	case err == nil:
		operation.InputCreator = creator
		return operation, 0, nil // go to next step
	case kebError.IsTemporaryError(err):
		log.Errorf("cannot create upgrade shoot input creator at the moment for plan %s: %s", operation.ProvisioningParameters.PlanID, err)
		return s.operationManager.RetryOperation(operation, "error while creating upgrade shoot input creator", err, 5*time.Second, 1*time.Minute, log)
	default:
		log.Errorf("cannot create input creator for plan %s: %s", operation.ProvisioningParameters.PlanID, err)
		return s.operationManager.OperationFailed(operation, "cannot create provisioning input creator", err, log)
	}
}
