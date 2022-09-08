package deprovisioning

import (
	"fmt"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"time"
)

type InitStep struct {
	operationManager *process.OperationManager
	operationTimeout time.Duration
	operationStorage storage.Operations
	instanceStorage  storage.Instances
}

func NewInitStep(operations storage.Operations, instances storage.Instances, operationTimeout time.Duration) *InitStep {
	return &InitStep{
		operationManager: process.NewOperationManager(operations),
		operationTimeout: operationTimeout,
		operationStorage: operations,
		instanceStorage:  instances,
	}
}

func (s *InitStep) Name() string {
	return "Initialisation"
}

func (s *InitStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if time.Since(operation.CreatedAt) > s.operationTimeout {
		log.Infof("operation has reached the time limit: operation was created at: %s", operation.CreatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", s.operationTimeout), nil, log)
	}

	if operation.State != orchestration.Pending {
		return operation, 0, nil
	}
	// Check concurrent operation
	lastOp, err := s.operationStorage.GetLastOperation(operation.InstanceID)
	if err != nil {
		return operation, time.Minute, nil
	}
	if !lastOp.IsFinished() {
		log.Infof("waiting for %s operation (%s) to be finished", lastOp.Type, lastOp.ID)
		return operation, time.Minute, nil
	}

	// read the instance details (it could happen that created deprovisioning operation has outdated one)
	instance, err := s.instanceStorage.GetByID(operation.InstanceID)
	if err != nil {
		if dberr.IsNotFound(err) {
			log.Warnf("the instance already deprovisioned")
			return s.operationManager.OperationFailed(operation, "the instance was already deprovisioned", err, log)
		}
		return operation, time.Second, nil
	}

	log.Info("Setting state 'in progress' and refreshing instance details")
	opr, delay, err := s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.State = domain.InProgress
		op.InstanceDetails = instance.InstanceDetails
		op.ProvisioningParameters.ErsContext = internal.InheritMissingERSContext(op.ProvisioningParameters.ErsContext, lastOp.ProvisioningParameters.ErsContext)
	}, log)
	if delay != 0 {
		log.Errorf("unable to update the operation (move to 'in progress'), retrying")
		return operation, delay, nil
	}

	return opr, 0, nil
}
