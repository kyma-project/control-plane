package deprovisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type RemoveInstanceStep struct {
	operationManager *process.OperationManager
	instanceStorage  storage.Instances
	operationStorage storage.Operations
}

var _ process.Step = &RemoveInstanceStep{}

func NewRemoveInstanceStep(instanceStorage storage.Instances, operationStorage storage.Operations) RemoveInstanceStep {
	return RemoveInstanceStep{
		operationManager: process.NewOperationManager(operationStorage),
		instanceStorage:  instanceStorage,
		operationStorage: operationStorage,
	}
}

func (s RemoveInstanceStep) Name() string {
	return "Remove_Instance"
}

func (s RemoveInstanceStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {

	if operation.Temporary {
		err := s.removeRuntimeID(operation, log)
		if err != nil {
			return operation, time.Second, err
		}
	} else {
		log.Info("Removing the instance")
		delay := s.removeInstancePermanently(operation.InstanceID)
		if delay != 0 {
			return operation, delay, nil
		}

		log.Info("Removing the userID field from operation")
		operation, delay, _ = s.operationManager.UpdateOperation(operation, func(operation *internal.Operation) {
			operation.ProvisioningParameters.ErsContext.UserID = ""
		}, log)
		if delay != 0 {
			return operation, delay, nil
		}

		return operation, 0, nil
	}
	return operation, 0, nil
}

func (s RemoveInstanceStep) removeInstancePermanently(instanceID string) time.Duration {
	err := s.instanceStorage.Delete(instanceID)
	if err != nil {
		return 10 * time.Second
	}

	return 0
}

func (s *RemoveInstanceStep) removeRuntimeID(operation internal.Operation, log logrus.FieldLogger) error {
	inst, err := s.instanceStorage.GetByID(operation.InstanceID)
	if err != nil {
		log.Errorf("cannot fetch instance with ID: %s from storage", operation.InstanceID)
		return err
	}

	// empty RuntimeID means there is no runtime in the Provisioner Domain
	inst.RuntimeID = ""
	_, err = s.instanceStorage.Update(*inst)
	if err != nil {
		log.Errorf("cannot update instance with ID: %s", inst.InstanceID)
		return err
	}

	operation1, err := s.operationStorage.GetOperationByID(operation.ID)
	if err != nil {
		log.Errorf("cannot get deprovisioning operation with ID: %s from storage", operation1.ID)
		return err
	}

	operation1.RuntimeID = ""
	_, err = s.operationStorage.UpdateOperation(*operation1)
	if err != nil {
		log.Errorf("cannot update deprovisioning operation with ID: %s", operation1.ID)
		return err
	}

	return nil
}
