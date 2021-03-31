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

const (
	ConnOfferingName = "connectivity"
	ConnPlanName     = "connectivity_proxy"
)

type ConnUpgradeProvisionStep struct {
	operationManager *process.UpgradeKymaOperationManager
}

func NewConnUpgradeProvisionStep(os storage.Operations) *ConnUpgradeProvisionStep {
	return &ConnUpgradeProvisionStep{
		operationManager: process.NewUpgradeKymaOperationManager(os),
	}
}

var _ Step = (*ConnUpgradeProvisionStep)(nil)

func (s *ConnUpgradeProvisionStep) Name() string {
	return "CONN_UpgradeProvision"
}

func (s *ConnUpgradeProvisionStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if operation.Conn.Instance.InstanceID != "" {
		log.Infof("Connectivity Upgrade-Provision was already done")
		return operation, 0, nil
	}
	if operation.Conn.Instance.ProvisioningTriggered {
		log.Infof("Conn Provisioning step was already triggered")
		return operation, 0, nil
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manage client"))
	}

	if operation.Conn.Instance.InstanceID == "" {
		op, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.UpgradeKymaOperation) {
			operation.Conn.Instance.InstanceID = uuid.New().String()
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
		operation.Conn.Instance.ProvisioningTriggered = true
	}, log)
	if retry > 0 {
		log.Errorf("unable to update operation")
		return operation, time.Second, nil
	}

	return operation, 0, nil
}

func (s *ConnUpgradeProvisionStep) provision(smCli servicemanager.Client, operation internal.UpgradeKymaOperation,
	log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {

	input := provisioning.GetConnProvisioningData(operation.Conn)
	resp, err := smCli.Provision(operation.Conn.Instance.BrokerID, *input, true)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("Provision() call failed for brokerID: %s; input: %#v", operation.Conn.Instance.BrokerID, input))
	}
	log.Debugf("response from Conn provisioning call: %#v", resp)

	return operation, 0, nil
}

func (s *ConnUpgradeProvisionStep) handleError(operation internal.UpgradeKymaOperation, err error, log logrus.FieldLogger, msg string) (internal.UpgradeKymaOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg, log)
}
