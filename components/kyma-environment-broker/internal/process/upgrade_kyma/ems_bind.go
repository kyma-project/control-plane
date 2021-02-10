package upgrade_kyma

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type EmsUpgradeBindStep struct {
	operationManager *process.UpgradeKymaOperationManager
	secretKey        string
}

func NewEmsUpgradeBindStep(os storage.Operations, secretKey string) *EmsUpgradeBindStep {
	return &EmsUpgradeBindStep{
		operationManager: process.NewUpgradeKymaOperationManager(os),
		secretKey:        secretKey,
	}
}

var _ Step = (*EmsUpgradeBindStep)(nil)

func (s *EmsUpgradeBindStep) Name() string {
	return "EMS_UpgradeBind"
}

func (s *EmsUpgradeBindStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if operation.Ems.BindingID != "" {
		log.Infof("Ems Upgrade-Bind was already done")
		return operation, 0, nil
	}
	if !operation.Ems.Instance.ProvisioningTriggered {
		return s.handleError(operation, fmt.Errorf("Ems Provisioning step was not triggered"), log, "")
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manage client"))
	}
	// test if the provisioning is finished, if not, retry after 10s
	resp, err := smCli.LastInstanceOperation(operation.Ems.Instance.InstanceKey(), "")
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("LastInstanceOperation() call failed"))
	}
	log.Infof("Provisioning Ems (instanceID=%s) state: %s", operation.Ems.Instance.InstanceID, resp.State)
	switch resp.State {
	case servicemanager.InProgress:
		return operation, 10 * time.Second, nil
	case servicemanager.Failed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("Ems provisioning failed: %s", resp.Description))
	}
	// execute binding
	var eventingOverrides *provisioning.EventingOverrides
	if !operation.Ems.Instance.Provisioned {
		if operation.Ems.BindingID == "" {
			operation.Ems.BindingID = uuid.New().String()
		}
		respBinding, err := smCli.Bind(operation.Ems.Instance.InstanceKey(), operation.Ems.BindingID, nil, false)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("Bind() call failed"))
		}
		// get overrides
		eventingOverrides, err = provisioning.GetEventingCredentials(respBinding.Binding)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("getCredentials() call failed"))
		}
		encryptedOverrides, err := provisioning.EncryptEventingOverrides(s.secretKey, eventingOverrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("encryptOverrides() call failed"))
		}
		operation.Ems.Overrides = encryptedOverrides
		operation.Ems.Instance.Provisioned = true
		operation.Ems.Instance.ProvisioningTriggered = false
		// save the status
		op, retry := s.operationManager.UpdateOperation(operation)
		if retry > 0 {
			log.Errorf("unable to update operation")
			return operation, time.Second, nil
		}
		operation = op
	} else {
		// get the credentials from encrypted string in operation.Ems.Instance.
		eventingOverrides, err = provisioning.DecryptEventingOverrides(s.secretKey, operation.Ems.Overrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("decryptOverrides() call failed"))
		}
	}

	// append overrides
	operation.InputCreator.AppendOverrides(components.Eventing, provisioning.GetEventingOverrides(eventingOverrides))

	return operation, 0, nil
}

func (s *EmsUpgradeBindStep) handleError(operation internal.UpgradeKymaOperation, err error, log logrus.FieldLogger, msg string) (internal.UpgradeKymaOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg)
}
