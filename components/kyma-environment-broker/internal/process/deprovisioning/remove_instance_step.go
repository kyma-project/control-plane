package deprovisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"

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

func NewRemoveInstanceStep(instanceStorage storage.Instances, operationStorage storage.Operations) *RemoveInstanceStep {
	return &RemoveInstanceStep{
		operationManager: process.NewOperationManager(operationStorage),
		instanceStorage:  instanceStorage,
		operationStorage: operationStorage,
	}
}

func (s *RemoveInstanceStep) Name() string {
	return "Remove_Instance"
}

func (s *RemoveInstanceStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	var backoff time.Duration

	_, err := s.instanceStorage.GetByID(operation.InstanceID)
	switch {
	case err == nil:
	case dberr.IsNotFound(err):
		log.Infof("instance already deleted", err)
		return operation, 0 * time.Second, nil
	default:
		log.Errorf("unable to get instance from the storage: %s", err)
		return operation, 1 * time.Second, nil
	}

	if operation.Temporary {
		log.Info("Removing the RuntimeID field from the instance")
		backoff = s.removeRuntimeIDFromInstance(operation.InstanceID, log)
		if backoff != 0 {
			return operation, backoff, nil
		}

		log.Info("Removing the RuntimeID field from the operation")
		operation, backoff, _ = s.operationManager.UpdateOperation(operation, func(operation *internal.Operation) {
			operation.RuntimeID = ""
		}, log)
	} else {
		log.Info("Removing the instance permanently")
		backoff = s.removeInstancePermanently(operation.InstanceID, log)
		if backoff != 0 {
			return operation, backoff, nil
		}

		log.Info("Removing the userID field from the operation")
		operation, backoff, _ = s.operationManager.UpdateOperation(operation, func(operation *internal.Operation) {
			operation.ProvisioningParameters.ErsContext.UserID = ""
		}, log)
	}

	return operation, backoff, nil
}

func (s RemoveInstanceStep) removeRuntimeIDFromInstance(instanceID string, log logrus.FieldLogger) time.Duration {
	backoff := time.Second

	instance, err := s.instanceStorage.GetByID(instanceID)
	if err != nil {
		log.Errorf("unable to get instance %s from the storage: %s", instanceID, err)
		return backoff
	}

	// empty RuntimeID means there is no runtime in the Provisioner Domain
	instance.RuntimeID = ""
	_, err = s.instanceStorage.Update(*instance)
	if err != nil {
		log.Errorf("unable to update instance %s in the storage: %s", instanceID, err)
		return backoff
	}

	return 0
}

func (s RemoveInstanceStep) removeInstancePermanently(instanceID string, log logrus.FieldLogger) time.Duration {
	err := s.instanceStorage.Delete(instanceID)
	if err != nil {
		log.Errorf("unable to remove instance %s from the storage: %s", instanceID, err)
		return 10 * time.Second
	}

	return 0
}
