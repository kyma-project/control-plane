package provisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/connectivity_bind"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type ConnectivityBindStep struct {
	operationManager *process.ProvisionOperationManager
	secretKey        string
}

func NewConnectivityBindStep(os storage.Operations, secretKey string) *ConnectivityBindStep {
	return &ConnectivityBindStep{
		operationManager: process.NewProvisionOperationManager(os),
		secretKey:        secretKey,
	}
}

var _ Step = (*ConnectivityBindStep)(nil)

func (s *ConnectivityBindStep) Name() string {
	return "Connectivity_Bind"
}

func (s *ConnectivityBindStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if !operation.Connectivity.Instance.ProvisioningTriggered {
		return s.handleError(operation, fmt.Errorf("connectivity Provisioning step was not triggered"), log, "")
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
	var connectivityOverrides *connectivity_bind.ConnectivityConfig
	if !operation.Connectivity.Instance.Provisioned {
		if operation.Connectivity.BindingID == "" {
			operation.Connectivity.BindingID = uuid.New().String()
		}
		respBinding, err := smCli.Bind(operation.Connectivity.Instance.InstanceKey(), operation.Connectivity.BindingID, nil, false)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("Bind() call failed"))
		}
		// get overrides
		connectivityOverrides, err = connectivity_bind.GetConnectivityCredentials(respBinding.Binding)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("unable to load config"))
		}
		encryptedOverrides, err := connectivity_bind.EncryptConnectivityConfig(s.secretKey, connectivityOverrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("unable to encrypt config"))
		}

		// save the status
		op, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
			operation.Connectivity.Overrides = encryptedOverrides
			operation.Connectivity.Instance.Provisioned = true
			operation.Connectivity.Instance.ProvisioningTriggered = false
		}, log)
		if retry > 0 {
			log.Errorf("unable to update operation")
			return operation, time.Second, nil
		}
		operation = op
	} else {
		// get the credentials from encrypted string in operation.Connectivity.Instance.
		connectivityOverrides, err = connectivity_bind.DecryptConnectivityConfig(s.secretKey, operation.Connectivity.Overrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("unable to decrypt configs"))
		}
		applyOverrides(connectivityOverrides, operation)
	}

	log.Debugf("Got Connectivity Service credentials from the binding.")

	return operation, 0, nil
}

func applyOverrides(connectivityOverrides *connectivity_bind.ConnectivityConfig, operation internal.ProvisioningOperation) {
	overrides := connectivity_bind.PrepareOverrides(connectivityOverrides)
	operation.InputCreator.AppendOverrides(connectivity_bind.ConnectivityProxyComponentName, overrides)
}

func (s *ConnectivityBindStep) handleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg, log)
}
