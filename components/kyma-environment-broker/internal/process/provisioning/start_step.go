package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

// StartStep changes the state from pending to in progress if necessary
type StartStep struct {
	operationStorage storage.Operations
	instanceStorage  storage.Instances
	operationManager *process.ProvisionOperationManager
}

func NewStartStep(os storage.Operations, is storage.Instances) *StartStep {
	return &StartStep{
		operationStorage: os,
		instanceStorage:  is,
		operationManager: process.NewProvisionOperationManager(os),
	}
}

func (s *StartStep) Name() string {
	return "Starting"
}

func (s *StartStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if operation.State != orchestration.Pending {
		return operation, 0, nil
	}

	deprovisionOp, err := s.operationStorage.GetDeprovisioningOperationByInstanceID(operation.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		log.Errorf("Unable to get deprovisioning operation: %s", err.Error())
		return operation, time.Second, nil
	}
	if deprovisionOp != nil && deprovisionOp.State == domain.InProgress {
		return operation, time.Minute, nil
	}

	// if there was a deprovisioning process before, take new InstanceDetails
	if deprovisionOp != nil {
		inst, err := s.instanceStorage.GetByID(operation.InstanceID)
		if err != nil {
			if dberr.IsNotFound(err) {
				log.Errorf("Instance does not exists.")
				return s.operationManager.OperationFailed(operation, "The instance does not exists", err, log)
			}
			log.Errorf("Unable to get the instance: %s", err.Error())
			return operation, time.Second, nil
		}
		log.Infof("Setting the newest InstanceDetails")
		operation.InstanceDetails, err = inst.GetInstanceDetails()
		if err != nil {
			log.Errorf("Unable to provide Instance details: %s", err.Error())
			return s.operationManager.OperationFailed(operation, "Unable to provide Instance details", err, log)
		}
	}
	lastOp, err := s.operationStorage.GetLastOperation(operation.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		log.Warn("Failed to get last operation for ERSContext:", err)
		return operation, time.Minute, nil
	}
	log.Infof("Setting the operation to 'InProgress'")
	newOp, retry, _ := s.operationManager.UpdateOperation(operation, func(op *internal.ProvisioningOperation) {
		if lastOp != nil {
			op.ProvisioningParameters.ErsContext = internal.InheritMissingERSContext(op.ProvisioningParameters.ErsContext, lastOp.ProvisioningParameters.ErsContext)
		}
		op.State = domain.InProgress
	}, log)
	operation = newOp
	if retry > 0 {
		return operation, time.Second, nil
	}

	return operation, 0, nil
}
