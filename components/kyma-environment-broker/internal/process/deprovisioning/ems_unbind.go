package deprovisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type EmsUnbindStep struct {
	operationManager *process.DeprovisionOperationManager
}

func NewEmsUnbindStep(os storage.Operations) *EmsUnbindStep {
	return &EmsUnbindStep{
		operationManager: process.NewDeprovisionOperationManager(os),
	}
}

var _ Step = (*EmsUnbindStep)(nil)

func (s *EmsUnbindStep) Name() string {
	return "EMS_Unbind"
}

func (s *EmsUnbindStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if operation.Ems.BindingID == "" {
		log.Infof("Ems Unbind step skipped, instance not bound")
		return operation, 0, nil
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manage client"))
	}

	// Unbind
	log.Infof("unbinding for EMS instance: %s started; binding: %s", operation.Ems.Instance.InstanceID, operation.Ems.BindingID)
	_, err = smCli.Unbind(operation.Ems.Instance.InstanceKey(), operation.Ems.BindingID, true)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to unbind, bindingId=%s", operation.Ems.BindingID))
	}
	log.Infof("unbinding for EMS instance: %s finished", operation.Ems.Instance.InstanceID)

	updatedOperation, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.DeprovisioningOperation) {
		operation.Ems.BindingID = ""
		operation.Ems.Overrides = ""
	}, log)
	return updatedOperation, retry, nil
}

func (s *EmsUnbindStep) handleError(operation internal.DeprovisioningOperation, err error, log logrus.FieldLogger,
	msg string) (internal.DeprovisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg, log)
}
