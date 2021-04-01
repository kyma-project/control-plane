package deprovisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type ConnectivityDeprovisionStep struct {
	operationManager *process.DeprovisionOperationManager
}

func NewConnectivityDeprovisionStep(os storage.Operations) *ConnectivityDeprovisionStep {
	return &ConnectivityDeprovisionStep{
		operationManager: process.NewDeprovisionOperationManager(os),
	}
}

var _ Step = (*ConnectivityDeprovisionStep)(nil)

func (s *ConnectivityDeprovisionStep) Name() string {
	return "CONN_Deprovision"
}

func (s *ConnectivityDeprovisionStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (
	internal.DeprovisioningOperation, time.Duration, error) {
	if operation.Connectivity.Instance.InstanceID == "" {
		log.Infof("Connectivity Deprovision step skipped, instance not provisioned")
		return operation, 0, nil
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manage client"))
	}

	log.Infof("deprovisioning for Connectivity instance: %s started", operation.Connectivity.Instance.InstanceID)
	_, err = smCli.Deprovision(operation.Connectivity.Instance.InstanceKey(), false)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("Deprovision() call failed"))
	}
	log.Infof("deprovisioning for Connectivity instance: %s finished", operation.Connectivity.Instance.InstanceID)

	updatedOperation, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.DeprovisioningOperation) {
		operation.Connectivity.Instance.InstanceID = ""
		operation.Connectivity.Instance.Provisioned = false
	}, log)
	return updatedOperation, retry, nil
}

func (s *ConnectivityDeprovisionStep) handleError(operation internal.DeprovisioningOperation, err error, log logrus.FieldLogger,
	msg string) (internal.DeprovisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg, log)
}
