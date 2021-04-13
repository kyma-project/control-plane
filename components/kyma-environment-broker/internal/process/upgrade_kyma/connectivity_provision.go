package upgrade_kyma

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type ConnectivityUpgradeProvisionStep struct {
	operationManager *process.UpgradeKymaOperationManager
}

func NewConnectivityUpgradeProvisionStep(os storage.Operations) *ConnectivityUpgradeProvisionStep {
	return &ConnectivityUpgradeProvisionStep{
		operationManager: process.NewUpgradeKymaOperationManager(os),
	}
}

var _ Step = (*ConnectivityUpgradeProvisionStep)(nil)

func (s *ConnectivityUpgradeProvisionStep) Name() string {
	return "Connectivity_UpgradeProvision"
}

func (s *ConnectivityUpgradeProvisionStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if operation.Connectivity.Instance.InstanceID != "" {
		log.Infof("Connectivity Upgrade-Provision was already done")
		return operation, 0, nil
	}
	if operation.Connectivity.Instance.ProvisioningTriggered {
		log.Infof("Connectivity Provisioning step was already triggered")
		return operation, 0, nil
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manager client"))
	}

	if operation.Connectivity.Instance.InstanceID == "" {
		op, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.UpgradeKymaOperation) {
			operation.Connectivity.Instance.InstanceID = uuid.New().String()
		}, log)
		if retry > 0 {
			log.Errorf("unable to update operation")
			return operation, time.Second, nil
		}
		operation = op
	}

	// provision
	operation, _, err = s.provision(smCli, operation, log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("provision()  call failed"))
	}
	// save the status
	operation, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.UpgradeKymaOperation) {
		operation.Connectivity.Instance.ProvisioningTriggered = true
	}, log)
	if retry > 0 {
		log.Errorf("unable to update operation")
		return operation, time.Second, nil
	}

	return operation, 0, nil
}

func (s *ConnectivityUpgradeProvisionStep) provision(smCli servicemanager.Client, operation internal.UpgradeKymaOperation,
	log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {

	input := provisioning.GetConnectivityProvisioningData(operation.Connectivity.Instance)
	resp, err := smCli.Provision(operation.Connectivity.Instance.BrokerID, *input, true)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("Provision() call failed for brokerID: %s; input: %#v", operation.Connectivity.Instance.BrokerID, input))
	}
	log.Debugf("response from Connectivity provisioning call: %#v", resp)

	return operation, 0, nil
}

func (s *ConnectivityUpgradeProvisionStep) handleError(operation internal.UpgradeKymaOperation, err error, log logrus.FieldLogger, msg string) (internal.UpgradeKymaOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg, log)
}
