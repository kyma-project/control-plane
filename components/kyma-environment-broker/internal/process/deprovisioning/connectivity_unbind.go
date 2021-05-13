package deprovisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type ConnectivityUnbindStep struct {
	operationManager *process.DeprovisionOperationManager
}

func NewConnectivityUnbindStep(os storage.Operations) *ConnectivityUnbindStep {
	return &ConnectivityUnbindStep{
		operationManager: process.NewDeprovisionOperationManager(os),
	}
}

var _ Step = (*ConnectivityUnbindStep)(nil)

func (s *ConnectivityUnbindStep) Name() string {
	return "Connectivity_Unbind"
}

func (s *ConnectivityUnbindStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if operation.Connectivity.BindingID == "" {
		log.Infof("Connectivity Unbind step skipped, instance not bound")
		return operation, 0, nil
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manager client"))
	}

	// Unbind
	log.Infof("unbinding for Connectivity instance: %s started; binding: %s", operation.Connectivity.Instance.InstanceID, operation.Connectivity.BindingID)
	_, err = smCli.Unbind(operation.Connectivity.Instance.InstanceKey(), operation.Connectivity.BindingID, true)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to unbind, bindingId=%s", operation.Connectivity.BindingID))
	}
	log.Infof("unbinding for Connectivity instance: %s finished", operation.Connectivity.Instance.InstanceID)

	updatedOperation, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.DeprovisioningOperation) {
		operation.Connectivity.BindingID = ""
		operation.Connectivity.Overrides = ""
	}, log)
	return updatedOperation, retry, nil
}

func (s *ConnectivityUnbindStep) handleError(operation internal.DeprovisioningOperation, err error, log logrus.FieldLogger,
	msg string) (internal.DeprovisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg, log)
}
