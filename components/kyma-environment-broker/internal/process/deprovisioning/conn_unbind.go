package deprovisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type ConnUnbindStep struct {
	operationManager *process.DeprovisionOperationManager
}

func NewConnUnbindStep(os storage.Operations) *ConnUnbindStep {
	return &ConnUnbindStep{
		operationManager: process.NewDeprovisionOperationManager(os),
	}
}

var _ Step = (*ConnUnbindStep)(nil)

func (s *ConnUnbindStep) Name() string {
	return "CONN_Unbind"
}

func (s *ConnUnbindStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if operation.Conn.BindingID == "" {
		log.Infof("Connectivity Unbind step skipped, instance not bound")
		return operation, 0, nil
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manage client"))
	}

	// Unbind
	log.Infof("unbinding for Connectivity instance: %s started; binding: %s", operation.Conn.Instance.InstanceID, operation.Conn.BindingID)
	_, err = smCli.Unbind(operation.Conn.Instance.InstanceKey(), operation.Conn.BindingID, true)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to unbind, bindingId=%s", operation.Conn.BindingID))
	}
	log.Infof("unbinding for Connectivity instance: %s finished", operation.Conn.Instance.InstanceID)

	updatedOperation, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.DeprovisioningOperation) {
		operation.Conn.BindingID = ""
		operation.Conn.Overrides = ""
	}, log)
	return updatedOperation, retry, nil
}

func (s *ConnUnbindStep) handleError(operation internal.DeprovisioningOperation, err error, log logrus.FieldLogger,
	msg string) (internal.DeprovisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg, log)
}
