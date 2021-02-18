package upgrade_kyma

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type EmsUpgradeProvisionStep struct {
	operationManager *process.UpgradeKymaOperationManager
}

func NewEmsUpgradeProvisionStep(os storage.Operations) *EmsUpgradeProvisionStep {
	return &EmsUpgradeProvisionStep{
		operationManager: process.NewUpgradeKymaOperationManager(os),
	}
}

var _ Step = (*EmsUpgradeProvisionStep)(nil)

func (s *EmsUpgradeProvisionStep) Name() string {
	return "EMS_UpgradeProvision"
}

func (s *EmsUpgradeProvisionStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if operation.Ems.Instance.InstanceID != "" {
		log.Infof("Ems Upgrade-Provision was already done")
		return operation, 0, nil
	}
	if operation.Ems.Instance.ProvisioningTriggered {
		log.Infof("Ems Upgrade-Provisioning step was already triggered")
		return operation, 0, nil
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manage client"))
	}

	// provision
	operation, _, err = s.provision(smCli, operation, log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("provision()  call failed"))
	}
	// save the status
	operation.Ems.Instance.ProvisioningTriggered = true
	operation, retry := s.operationManager.UpdateOperation(operation)
	if retry > 0 {
		log.Errorf("unable to update operation")
		return operation, time.Second, nil
	}

	return operation, 0, nil
}

func (s *EmsUpgradeProvisionStep) provision(smCli servicemanager.Client, operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	input := provisioning.GetEventingProvisioningData(operation.Ems)
	resp, err := smCli.Provision(operation.Ems.Instance.BrokerID, *input, true)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("Provision() call failed for brokerID: %s; input: %#v", operation.Ems.Instance.BrokerID, input))
	}
	log.Debugf("response from EMS provisioning call: %#v", resp)

	operation.Ems.Instance.InstanceID = input.ID

	return operation, 0, nil
}

func (s *EmsUpgradeProvisionStep) handleError(operation internal.UpgradeKymaOperation, err error, log logrus.FieldLogger, msg string) (internal.UpgradeKymaOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg)
}
