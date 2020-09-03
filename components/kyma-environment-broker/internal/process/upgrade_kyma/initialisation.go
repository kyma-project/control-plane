package upgrade_kyma

import (
	"errors"
	"fmt"
	"time"

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
	operationManager  *process.UpgradeKymaOperationManager
	operationStorage  storage.Provisioning
	instanceStorage   storage.Instances
	provisionerClient provisioner.Client

	desiredKymaVersion string
}

func NewInitialisationStep(os storage.Operations, is storage.Instances, pc provisioner.Client) *InitialisationStep {
	return &InitialisationStep{
		operationManager:  process.NewUpgradeKymaOperationManager(os),
		operationStorage:  os,
		instanceStorage:   is,
		provisionerClient: pc,
	}
}

func (s *InitialisationStep) Name() string {
	return "Upgrade_Kyma_Initialisation"
}

func (s *InitialisationStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	// rewrite necessary data from ProvisioningOperation to operation internal.UpgradeOperation
	op, err := s.operationStorage.GetProvisioningOperationByInstanceID(operation.InstanceID)
	if err != nil {
		log.Errorf("while getting provisioning operation from storage")
		return operation, time.Second * 10, nil
	}
	if op.State == domain.InProgress {
		log.Info("waiting for provisioning operation to finish")
		return operation, time.Minute, nil
	}

	parameters, err := op.GetProvisioningParameters()
	if err != nil {
		return s.operationManager.OperationFailed(operation, "cannot get provisioning parameters from operation")
	}

	err = operation.SetProvisioningParameters(parameters)
	if err != nil {
		log.Error("Aborting after failing to save provisioning parameters for operation")
		return s.operationManager.OperationFailed(operation, err.Error())
	}

	instance, err := s.instanceStorage.GetByID(operation.InstanceID)
	switch {
	case err == nil:
		if operation.ProvisionerOperationID == "" {
			return operation, 0, nil // upgrade not started, go to next step
		}
		log.Infof("runtime being upgraded, check operation status")
		operation.RuntimeID = instance.RuntimeID
		return s.checkRuntimeStatus(operation, instance, log.WithField("runtimeID", instance.RuntimeID))
	case dberr.IsNotFound(err):
		return s.operationManager.OperationSucceeded(operation, "instance already deprovisioned")
	default:
		log.Errorf("unable to get instance from storage: %s", err)
		return operation, 1 * time.Second, nil
	}

}

func (s *InitialisationStep) initializeUpgradeRuntimeRequest(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	pp, err := operation.GetProvisioningParameters()
	if err != nil {
		log.Errorf("cannot fetch provisioning parameters from operation: %s", err)
		return s.operationManager.OperationFailed(operation, "invalid operation provisioning parameters")
	}

	log.Infof("create upgrade input creator for plan ID %q", pp.PlanID)
	return operation, time.Second, nil
}

func (s *InitialisationStep) getDesiredKymaVersion() (string, error) {
	if s.desiredKymaVersion == "" {
		return "", errors.New("desired Kyma version was not set")
	}
	return s.desiredKymaVersion, nil
}

func (s *InitialisationStep) checkRuntimeStatus(operation internal.UpgradeKymaOperation, instance *internal.Instance, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
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
		return s.operationManager.OperationSucceeded(operation, msg)
	case gqlschema.OperationStateInProgress:
		return operation, 1 * time.Minute, nil
	case gqlschema.OperationStatePending:
		return operation, 1 * time.Minute, nil
	case gqlschema.OperationStateFailed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("provisioner client returns failed status: %s", msg))
	}

	return s.operationManager.OperationFailed(operation, fmt.Sprintf("unsupported provisioner client status: %s", status.State.String()))
}
