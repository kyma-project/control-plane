package deprovisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type RemoveInstancePermanentlyStep struct {
	operationManager *process.OperationManager
	instanceStorage  storage.Instances
}

var _ process.Step = &RemoveInstancePermanentlyStep{}

func NewRemoveInstancePermanentlyStep(instanceStorage storage.Instances, operationStorage storage.Operations) RemoveInstancePermanentlyStep {
	return RemoveInstancePermanentlyStep{
		operationManager: process.NewOperationManager(operationStorage),
		instanceStorage:  instanceStorage,
	}
}

func (s RemoveInstancePermanentlyStep) Name() string {
	return "Remove_Instance_Permanently"
}

func (s RemoveInstancePermanentlyStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {

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

func (s RemoveInstancePermanentlyStep) removeInstancePermanently(instanceID string) time.Duration {
	err := s.instanceStorage.Delete(instanceID)
	if err != nil {
		return 10 * time.Second
	}

	return 0
}
