package deprovisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
)

const (
	// the time after which the operation is marked as expired
	CheckStatusTimeout = 5 * time.Hour
)

type InitialisationStep struct {
	operationManager  *process.DeprovisionOperationManager
	operationStorage  storage.Operations
	instanceStorage   storage.Instances
	provisionerClient provisioner.Client
	accountProvider   hyperscaler.AccountProvider
	operationTimeout  time.Duration
}

func NewInitialisationStep(os storage.Operations, is storage.Instances, pc provisioner.Client, accountProvider hyperscaler.AccountProvider, operationTimeout time.Duration) *InitialisationStep {
	return &InitialisationStep{
		operationManager:  process.NewDeprovisionOperationManager(os),
		operationStorage:  os,
		instanceStorage:   is,
		provisionerClient: pc,
		accountProvider:   accountProvider,
		operationTimeout:  operationTimeout,
	}
}

func (s *InitialisationStep) Name() string {
	return "Deprovision_Initialization"
}

func (s *InitialisationStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	op, when, err := s.run(operation, log)

	if op.State == domain.Succeeded {
		if op.Temporary {
			log.Info("Removing RuntimeID from the instance")
			err := s.removeRuntimeID(operation, log)
			if err != nil {
				return operation, time.Second, err
			}
		} else {
			log.Info("Removing the instance")
			repeat, err := s.removeInstance(operation.InstanceID)
			if err != nil || repeat != 0 {
				return operation, repeat, err
			}
			log.Info("Removing the userID field from operation")
			op, repeat = s.removeUserID(op, log)
			if repeat != 0 {
				return operation, repeat, nil
			}
		}
	}
	return op, when, err
}

func (s *InitialisationStep) run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if time.Since(operation.CreatedAt) > s.operationTimeout {
		log.Infof("operation has reached the time limit: operation was created at: %s", operation.CreatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", s.operationTimeout), nil, log)
	}

	// rewrite necessary data from ProvisioningOperation to operation internal.DeprovisioningOperation
	op, err := s.operationStorage.GetProvisioningOperationByInstanceID(operation.InstanceID)
	if err != nil {
		log.Errorf("while getting provisioning operation from storage")
		return operation, time.Second * 10, nil
	}
	if op.State == domain.InProgress {
		log.Info("waiting for provisioning operation to finish")
		// This is only in memory copy because metrics depend on provisioning parameters being available, this doesn't persist them in KEB database
		operation.SubAccountID = operation.ProvisioningParameters.ErsContext.SubAccountID
		operation.ProvisioningParameters = op.ProvisioningParameters
		return operation, time.Minute, nil
	}
	lastOp, err := s.operationStorage.GetLastOperation(operation.InstanceID)
	if err != nil {
		return operation, time.Minute, nil
	}
	operation, repeat, _ := s.operationManager.UpdateOperation(operation, func(operation *internal.DeprovisioningOperation) {
		setAvsIds(operation, op, log)
		operation.SubAccountID = operation.ProvisioningParameters.ErsContext.SubAccountID
		operation.ProvisioningParameters = op.ProvisioningParameters
		operation.ProvisioningParameters.ErsContext = internal.InheritMissingERSContext(operation.ProvisioningParameters.ErsContext, lastOp.ProvisioningParameters.ErsContext)
	}, log)
	if repeat != 0 {
		return operation, time.Second, nil
	}

	instance, err := s.instanceStorage.GetByID(operation.InstanceID)
	switch {
	case err == nil:
		if operation.State == orchestration.Pending {
			details, err := instance.GetInstanceDetails()
			if err != nil {
				return s.operationManager.OperationFailed(operation, "unable to provide instance details", err, log)
			}
			log.Info("Setting state 'in progress' and refreshing instance details")
			var retry time.Duration
			operation, retry, _ = s.operationManager.UpdateOperation(operation, func(operation *internal.DeprovisioningOperation) {
				operation.State = domain.InProgress
				operation.InstanceDetails = details
			}, log)
			if retry > 0 {
				return operation, retry, nil
			}
		}

		if operation.ProvisionerOperationID == "" {
			return operation, 0, nil
		}
		log.Info("runtime being removed, check operation status")
		operation.RuntimeID = instance.RuntimeID
		return s.checkRuntimeStatus(operation, instance, log.WithField("runtimeID", instance.RuntimeID))
	case dberr.IsNotFound(err):
		return s.operationManager.OperationSucceeded(operation, "instance already deprovisioned", log)
	default:
		log.Errorf("unable to get instance from storage: %s", err)
		return operation, 1 * time.Second, nil
	}
}

func setAvsIds(deprovisioningOperation *internal.DeprovisioningOperation, provisioningOperation *internal.ProvisioningOperation, logger logrus.FieldLogger) {
	logger.Infof("AVS data from provisioning operation is [%+v]", provisioningOperation.Avs)
	if deprovisioningOperation.Avs.AvsEvaluationInternalId == 0 {
		deprovisioningOperation.Avs.AvsEvaluationInternalId = provisioningOperation.Avs.AvsEvaluationInternalId
	}
	if deprovisioningOperation.Avs.AVSEvaluationExternalId == 0 {
		deprovisioningOperation.Avs.AVSEvaluationExternalId = provisioningOperation.Avs.AVSEvaluationExternalId
	}
}

func (s *InitialisationStep) checkRuntimeStatus(operation internal.DeprovisioningOperation, instance *internal.Instance, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > CheckStatusTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", CheckStatusTimeout), nil, log)
	}

	status, err := s.provisionerClient.RuntimeOperationStatus(instance.GlobalAccountID, operation.ProvisionerOperationID)
	if err != nil {
		log.Errorf("call to provisioner RuntimeOperationStatus failed: %s", err.Error())
		return operation, 1 * time.Minute, nil
	}
	log.Infof("call to provisioner returned %s status", status.State.String())

	var msg string
	if status.Message != nil {
		msg = *status.Message
	}

	planID := instance.ServicePlanID

	switch status.State {
	case gqlschema.OperationStateSucceeded:
		{
			if !broker.IsTrialPlan(planID) {
				hypType, err := hyperscaler.FromCloudProvider(instance.Provider)
				if err != nil {
					log.Errorf("after successful deprovisioning failing to hyperscaler release subscription - determine the type of Hyperscaler to use for planID [%s]: %s", planID, err.Error())
					return operation, 0, nil
				}

				err = s.accountProvider.MarkUnusedGardenerSecretBindingAsDirty(hypType, instance.GetSubscriptionGlobalAccoundID())
				if err != nil {
					log.Errorf("after successful deprovisioning failed to release hyperscaler subscription: %s", err)
					return operation, 10 * time.Second, nil
				}
			}
			return s.operationManager.OperationSucceeded(operation, msg, log)
		}

	case gqlschema.OperationStateInProgress:
		return operation, 1 * time.Minute, nil
	case gqlschema.OperationStatePending:
		return operation, 1 * time.Minute, nil
	case gqlschema.OperationStateFailed:
		lastErr := provisioner.OperationStatusLastError(status.LastError)
		return s.operationManager.OperationFailed(operation, "provisioner client returns failed status", lastErr, log)
	}

	lastErr := provisioner.OperationStatusLastError(status.LastError)
	return s.operationManager.OperationFailed(operation, fmt.Sprintf("unsupported provisioner client status: %s", status.State.String()), lastErr, log)
}

func (s *InitialisationStep) removeInstance(instanceID string) (time.Duration, error) {
	err := s.instanceStorage.Delete(instanceID)
	if err != nil {
		return 10 * time.Second, nil
	}

	return 0, nil
}

func (s *InitialisationStep) removeUserID(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration) {
	updatedOperation, delay, _ := s.operationManager.UpdateOperation(operation, func(operation *internal.DeprovisioningOperation) {
		operation.ProvisioningParameters.ErsContext.UserID = ""
	}, log)

	return updatedOperation, delay
}

func (s *InitialisationStep) removeRuntimeID(op internal.DeprovisioningOperation, log logrus.FieldLogger) error {
	inst, err := s.instanceStorage.GetByID(op.InstanceID)
	if err != nil {
		log.Errorf("cannot fetch instance with ID: %s from storage", op.InstanceID)
		return err
	}

	// empty RuntimeID means there is no runtime in the Provisioner Domain
	inst.RuntimeID = ""
	_, err = s.instanceStorage.Update(*inst)
	if err != nil {
		log.Errorf("cannot update instance with ID: %s", inst.InstanceID)
		return err
	}

	operation, err := s.operationStorage.GetDeprovisioningOperationByID(op.ID)
	if err != nil {
		log.Errorf("cannot get deprovisioning operation with ID: %s from storage", op.ID)
		return err
	}

	operation.RuntimeID = ""
	_, err = s.operationStorage.UpdateDeprovisioningOperation(*operation)
	if err != nil {
		log.Errorf("cannot update deprovisioning operation with ID: %s", operation.ID)
		return err
	}

	return nil
}
