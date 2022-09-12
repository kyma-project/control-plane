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
	var delay time.Duration

	if operation.Temporary {
		log.Info("Removing the RuntimeID field from the instance")
		delay = s.removeRuntimeIDFromInstance(operation.InstanceID)
		if delay != 0 {
			return operation, delay, nil
		}

		log.Info("Removing the RuntimeID field from the operation")
		operation, delay, _ = s.operationManager.UpdateOperation(operation, func(operation *internal.Operation) {
			operation.RuntimeID = ""
		}, log)
	} else {
		log.Info("Removing the instance permanently")
		delay = s.removeInstancePermanently(operation.InstanceID)
		if delay != 0 {
			return operation, delay, nil
		}

		log.Info("Removing the userID field from the operation")
		operation, delay, _ = s.operationManager.UpdateOperation(operation, func(operation *internal.Operation) {
			operation.ProvisioningParameters.ErsContext.UserID = ""
		}, log)
	}

	return operation, delay, nil
}

func (s RemoveInstanceStep) removeRuntimeIDFromInstance(instanceID string) time.Duration {
	delay := time.Second

	instance, err := s.instanceStorage.GetByID(instanceID)
	if err != nil {
		return delay
	}

	// empty RuntimeID means there is no runtime in the Provisioner Domain
	instance.RuntimeID = ""
	_, err = s.instanceStorage.Update(*instance)
	if err != nil {
		return delay
	}

	return 0
}

func (s RemoveInstanceStep) removeInstancePermanently(instanceID string) time.Duration {
	err := s.instanceStorage.Delete(instanceID)
	if err != nil {
		return 10 * time.Second
	}

	return 0
}
