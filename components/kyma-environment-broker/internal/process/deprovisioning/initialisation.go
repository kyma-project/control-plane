package deprovisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/hyperscaler"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/sirupsen/logrus"
)

const (
	// the time after which the operation is marked as expired
	CheckStatusTimeout = 3 * time.Hour
)

type InitialisationStep struct {
	operationManager  *process.DeprovisionOperationManager
	operationStorage  storage.Provisioning
	instanceStorage   storage.Instances
	provisionerClient provisioner.Client
	accountProvider   hyperscaler.AccountProvider
}

func NewInitialisationStep(os storage.Operations, is storage.Instances, pc provisioner.Client, accountProvider hyperscaler.AccountProvider) *InitialisationStep {
	return &InitialisationStep{
		operationManager:  process.NewDeprovisionOperationManager(os),
		operationStorage:  os,
		instanceStorage:   is,
		provisionerClient: pc,
		accountProvider:   accountProvider,
	}
}

func (s *InitialisationStep) Name() string {
	return "Deprovision_Initialization"
}

func (s *InitialisationStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	op, when, err := s.run(operation, log)

	if op.State == domain.Succeeded {
		repeat, err := s.removeInstance(operation.InstanceID)
		if err != nil || repeat != 0 {
			return operation, repeat, err
		}
	}
	return op, when, err
}

func (s *InitialisationStep) run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	// rewrite necessary data from ProvisioningOperation to operation internal.DeprovisioningOperation
	op, err := s.operationStorage.GetProvisioningOperationByInstanceID(operation.InstanceID)
	if err != nil {
		log.Errorf("while getting provisioning operation from storage")
		return operation, time.Second * 10, nil
	}
	if op.State == domain.InProgress {
		log.Info("waiting for provisioning operation to finish")
		return operation, time.Minute, nil
	}

	setAvsIds(&operation, op, log)

	parameters, err := op.GetProvisioningParameters()
	if err != nil {
		return s.operationManager.OperationFailed(operation, "cannot get provisioning parameters from operation")
	}
	operation.SubAccountID = parameters.ErsContext.SubAccountID

	err = operation.SetProvisioningParameters(parameters)
	if err != nil {
		log.Error("Aborting after failing to save provisioning parameters for operation")
		return s.operationManager.OperationFailed(operation, err.Error())
	}

	instance, err := s.instanceStorage.GetByID(operation.InstanceID)
	switch {
	case err == nil:
		if operation.ProvisionerOperationID == "" {
			return operation, 0, nil
		}
		log.Info("instance being removed, check operation status")
		operation.RuntimeID = instance.RuntimeID
		return s.checkRuntimeStatus(operation, instance, parameters.PlanID, log.WithField("runtimeID", instance.RuntimeID))
	case dberr.IsNotFound(err):
		return s.operationManager.OperationSucceeded(operation, "instance already deprovisioned")
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

func (s *InitialisationStep) checkRuntimeStatus(operation internal.DeprovisioningOperation, instance *internal.Instance, planID string, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > CheckStatusTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", CheckStatusTimeout))
	}

	status, err := s.provisionerClient.RuntimeOperationStatus(instance.GlobalAccountID, operation.ProvisionerOperationID)
	if err != nil {
		return operation, 1 * time.Minute, nil
	}
	log.Infof("call to provisioner returned %s status", status.State.String())

	var msg string
	if status.Message != nil {
		msg = *status.Message
	}

	switch status.State {
	case gqlschema.OperationStateSucceeded:
		{

			// TODO:
			//After moving from POC into Production phase
			//Move the code retated to relesing subscription into the pool into separate step executed independently after runtime
			//is sucessfully  deprovisioned

			if !broker.IsTrialPlan(planID) {
				hypType, err := hyperscaler.HyperscalerTypeForPlanID(planID)
				if err != nil {
					log.Errorf("after successful deprovisioning failing to hyperscaler release subscription - determine the type of Hyperscaler to use for planID: %s", planID)
					return operation, 0, nil
				}

				err = s.accountProvider.ReleaseGardenerSecretForLastCluster(hypType, instance.GlobalAccountID)
				if err != nil {
					log.Errorf("after successful deprovisioning failed to release hyperscaler subscription: %s", err)
					return operation, 10 * time.Second, nil
				}
			}
			return s.operationManager.OperationSucceeded(operation, msg)
		}

	case gqlschema.OperationStateInProgress:
		return operation, 1 * time.Minute, nil
	case gqlschema.OperationStatePending:
		return operation, 1 * time.Minute, nil
	case gqlschema.OperationStateFailed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("provisioner client returns failed status: %s", msg))
	}

	return s.operationManager.OperationFailed(operation, fmt.Sprintf("unsupported provisioner client status: %s", status.State.String()))
}

func (s *InitialisationStep) removeInstance(instanceID string) (time.Duration, error) {
	err := s.instanceStorage.Delete(instanceID)
	if err != nil {
		return 10 * time.Second, nil
	}

	return 0, nil
}
