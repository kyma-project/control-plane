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

type ConnectivityUpgradeBindStep struct {
	operationManager *process.UpgradeKymaOperationManager
	secretKey        string
}

func NewConnectivityUpgradeBindStep(os storage.Operations, secretKey string) *ConnectivityUpgradeBindStep {
	return &ConnectivityUpgradeBindStep{
		operationManager: process.NewUpgradeKymaOperationManager(os),
		secretKey:        secretKey,
	}
}

var _ Step = (*ConnectivityUpgradeBindStep)(nil)

func (s *ConnectivityUpgradeBindStep) Name() string {
	return "Connectivity_UpgradeBind"
}

func (s *ConnectivityUpgradeBindStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if operation.Connectivity.BindingID != "" {
		log.Infof("Connectivity Upgrade-Bind was already done")
		return operation, 0, nil
	}
	if !operation.Connectivity.Instance.ProvisioningTriggered {
		return s.handleError(operation, fmt.Errorf("Connectivity Provisioning step was not triggered"), log, "")
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manager client"))
	}
	// test if the provisioning is finished, if not, retry after 10s
	resp, err := smCli.LastInstanceOperation(operation.Connectivity.Instance.InstanceKey(), "")
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("LastInstanceOperation() call failed"))
	}
	log.Infof("Provisioning Connectivity (instanceID=%s) state: %s", operation.Connectivity.Instance.InstanceID, resp.State)
	switch resp.State {
	case servicemanager.InProgress:
		return operation, 10 * time.Second, nil
	case servicemanager.Failed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("Connectivity provisioning failed: %s", resp.Description), log)
	}
	// execute binding
	var connectivityOverrides *provisioning.ConnectivityConfig
	if !operation.Connectivity.Instance.Provisioned {
		if operation.Connectivity.BindingID == "" {
			operation.Connectivity.BindingID = uuid.New().String()
		}
		respBinding, err := smCli.Bind(operation.Connectivity.Instance.InstanceKey(), operation.Connectivity.BindingID, nil, false)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("Bind() call failed"))
		}
		// get overrides
		connectivityOverrides, err = provisioning.GetConnectivityCredentials(respBinding.Binding)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("unable to load config"))
		}
		encryptedOverrides, err := provisioning.EncryptConnectivityConfig(s.secretKey, connectivityOverrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("unable to encrypt configs"))
		}
		// save the status
		op, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.UpgradeKymaOperation) {
			operation.Connectivity.Overrides = encryptedOverrides
			operation.Connectivity.Instance.Provisioned = true
			operation.Connectivity.Instance.ProvisioningTriggered = false
		}, log)
		if retry > 0 {
			return operation, time.Second, nil
		}
		operation = op
	} else {
		// get the credentials from encrypted string in operation.Connectivity.Instance.
		connectivityOverrides, err = provisioning.DecryptConnectivityConfig(s.secretKey, operation.Connectivity.Overrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("unable to decrypt configs"))
		}
	}

	// TODO: Decide how we want to pass this data to the SKR. Currently,
	//       credentials are prepared as a ConnectivityConfig structure.
	//       See the github card - https://github.com/orgs/kyma-project/projects/6#card-56776111
	//       ...
	//       - [ ] define what changes need to be done in KEB to
	//             allow passing secrets data to the Provisioner
	log.Debugf("Got Connectivity Service credentials from the binding.")

	return operation, 0, nil
}

func (s *ConnectivityUpgradeBindStep) handleError(operation internal.UpgradeKymaOperation, err error, log logrus.FieldLogger, msg string) (internal.UpgradeKymaOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg, log)
}
