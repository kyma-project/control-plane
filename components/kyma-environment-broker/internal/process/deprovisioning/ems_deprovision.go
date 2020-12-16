package deprovisioning

import (
<<<<<<< HEAD
	"time"

=======
	"fmt"
<<<<<<< HEAD
>>>>>>> 1b013b52... Use generic get offerings step
=======
	"time"

>>>>>>> ec1e40a0... Solve check-imports issues
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type EmsDeprovisionStep struct {
	operationManager *process.DeprovisionOperationManager
}

func NewEmsDeprovisionStep(os storage.Operations) *EmsDeprovisionStep {
	return &EmsDeprovisionStep{
		operationManager: process.NewDeprovisionOperationManager(os),
	}
}

var _ Step = (*EmsDeprovisionStep)(nil)

func (s *EmsDeprovisionStep) Name() string {
	return "EMS_Deprovision"
}

func (s *EmsDeprovisionStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (
	internal.DeprovisioningOperation, time.Duration, error) {
	if operation.Ems.Instance.InstanceID == "" {
		return operation, 0, nil
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("Step %s : unable to create Service Manage client", s.Name()))
	}

	log.Infof("Step %s : deprovisioning for EMS instance: %s started", s.Name(), operation.Ems.Instance.InstanceID)
	_, err = smCli.Deprovision(operation.Ems.Instance.InstanceKey(), false)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("Step %s : unable to deprovision", s.Name()))
	}
	log.Infof("Step %s : deprovisioning for EMS instance: %s finished", s.Name(), operation.Ems.Instance.InstanceID)

	operation.Ems.Instance.InstanceID = ""
	operation.Ems.Instance.Provisioned = false
	return s.operationManager.UpdateOperation(operation)
}

func (s *EmsDeprovisionStep) handleError(operation internal.DeprovisioningOperation, err error, log logrus.FieldLogger,
	msg string) (internal.DeprovisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg)
}
