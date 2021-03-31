package upgrade_kyma

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type ConnUpgradeBindStep struct {
	operationManager *process.UpgradeKymaOperationManager
	secretKey        string
}

func NewConnUpgradeBindStep(os storage.Operations, secretKey string) *ConnUpgradeBindStep {
	return &ConnUpgradeBindStep{
		operationManager: process.NewUpgradeKymaOperationManager(os),
		secretKey:        secretKey,
	}
}

var _ Step = (*ConnUpgradeBindStep)(nil)

func (s *ConnUpgradeBindStep) Name() string {
	return "CONN"
}

func (s *ConnUpgradeBindStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if operation.Conn.BindingID != "" {
		log.Infof("Connectivity Upgrade-Bind was already done")
		return operation, 0, nil
	}
	if !operation.Conn.Instance.ProvisioningTriggered {
		return s.handleError(operation, fmt.Errorf("Connectivity Provisioning step was not triggered"), log, "")
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manage client"))
	}
	// test if the provisioning is finished, if not, retry after 10s
	resp, err := smCli.LastInstanceOperation(operation.Conn.Instance.InstanceKey(), "")
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("LastInstanceOperation() call failed"))
	}
	log.Infof("Provisioning Connectivity (instanceID=%s) state: %s", operation.Conn.Instance.InstanceID, resp.State)
	switch resp.State {
	case servicemanager.InProgress:
		return operation, 10 * time.Second, nil
	case servicemanager.Failed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("Connectivity provisioning failed: %s", resp.Description), log)
	}
	// execute binding
	var connectivityOverrides *provisioning.ConnectivityOverrides
	if !operation.Conn.Instance.Provisioned {
		if operation.Conn.BindingID == "" {
			operation.Conn.BindingID = uuid.New().String()
		}
		respBinding, err := smCli.Bind(operation.Conn.Instance.InstanceKey(), operation.Conn.BindingID, nil, false)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("Bind() call failed"))
		}
		// get overrides
		connectivityOverrides, err = provisioning.GetConnCredentials(respBinding.Binding)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("getCredentials() call failed"))
		}
		encryptedOverrides, err := provisioning.EncryptConnOverrides(s.secretKey, connectivityOverrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("encryptOverrides() call failed"))
		}
		// save the status
		op, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.UpgradeKymaOperation) {
			operation.Conn.Overrides = encryptedOverrides
			operation.Conn.Instance.Provisioned = true
			operation.Conn.Instance.ProvisioningTriggered = false
		}, log)
		if retry > 0 {
			return operation, time.Second, nil
		}
		operation = op
	} else {
		// get the credentials from encrypted string in operation.Conn.Instance.
		connectivityOverrides, err = provisioning.DecryptConnOverrides(s.secretKey, operation.Conn.Overrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("decryptOverrides() call failed"))
		}
	}

	// TODO: Decide how we want to pass this data to the SKR.
	//       See the github card - https://github.com/orgs/kyma-project/projects/6#card-56776111
	//       ...
	//       - [ ] define what changes need to be done in KEB to
	//             allow passing secrets data to the Provisioner
	// append overrides
	//operation.InputCreator.AppendOverrides(components.Connectivity, provisioning.GetConnOverrides(connectivityOverrides))

	return operation, 0, nil
}

func (s *ConnUpgradeBindStep) handleError(operation internal.UpgradeKymaOperation, err error, log logrus.FieldLogger, msg string) (internal.UpgradeKymaOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg, log)
}
