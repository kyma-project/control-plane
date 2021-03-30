package deprovisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type ConnDeprovisionStep struct {
	operationManager *process.DeprovisionOperationManager
}

func NewConnDeprovisionStep(os storage.Operations) *ConnDeprovisionStep {
	return &ConnDeprovisionStep{
		operationManager: process.NewDeprovisionOperationManager(os),
	}
}

var _ Step = (*ConnDeprovisionStep)(nil)

func (s *ConnDeprovisionStep) Name() string {
	return "CONN_Deprovision"
}

func (s *ConnDeprovisionStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (
	internal.DeprovisioningOperation, time.Duration, error) {
	if operation.Conn.Instance.InstanceID == "" {
		log.Infof("Connectivity Deprovision step skipped, instance not provisioned")
		return operation, 0, nil
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manage client"))
	}

	log.Infof("deprovisioning for Connectivity instance: %s started", operation.Conn.Instance.InstanceID)
	_, err = smCli.Deprovision(operation.Conn.Instance.InstanceKey(), false)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("Deprovision() call failed"))
	}
	log.Infof("deprovisioning for Connectivity instance: %s finished", operation.Conn.Instance.InstanceID)

	updatedOperation, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.DeprovisioningOperation) {
		operation.Conn.Instance.InstanceID = ""
		operation.Conn.Instance.Provisioned = false
	}, log)
	return updatedOperation, retry, nil
}

func (s *ConnDeprovisionStep) handleError(operation internal.DeprovisioningOperation, err error, log logrus.FieldLogger,
	msg string) (internal.DeprovisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg, log)
}
