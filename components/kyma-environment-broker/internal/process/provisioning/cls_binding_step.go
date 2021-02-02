package provisioning

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"time"
)

type ClsBindStep struct {
	operationManager *process.ProvisionOperationManager
	secretKey        string
}

func NewClsBindStep(os storage.Operations, secretKey string) *ClsBindStep {
	return &ClsBindStep{
		operationManager: process.NewProvisionOperationManager(os),
		secretKey:        secretKey,
	}
}

var _ Step = (*ClsBindStep)(nil)

func (s *ClsBindStep) Name() string {
	return "CLS_Bind"
}

func (s *ClsBindStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if !operation.Cls.Instance.ProvisioningTriggered {
		return s.handleError(operation, fmt.Errorf("Ems Provisioning step was not triggered"), log, "")
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manage client"))
	}
	// test if the provisioning is finished, if not, retry after 10s
	resp, err := smCli.LastInstanceOperation(operation.Cls.Instance.InstanceKey(), "")
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("LastInstanceOperation() call failed"))
	}
	log.Infof("Provisioning Ems (instanceID=%s) state: %s", operation.Cls.Instance.InstanceID, resp.State)
	switch resp.State {
	case servicemanager.InProgress:
		return operation, 10 * time.Second, nil
	case servicemanager.Failed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("Ems provisioning failed: %s", resp.Description))
	}
	// execute binding
	var eventingOverrides *EventingOverrides
	if !operation.Cls.Instance.Provisioned {
		if operation.Cls.BindingID == "" {
			operation.Cls.BindingID = uuid.New().String()
		}
		respBinding, err := smCli.Bind(operation.Cls.Instance.InstanceKey(), operation.Cls.BindingID, nil, false)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("Bind() call failed"))
		}
		// get overrides
		eventingOverrides, err = getCredentials(respBinding.Binding)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("getCredentials() call failed"))
		}
		encryptedOverrides, err := encryptOverrides(s.secretKey, eventingOverrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("encryptOverrides() call failed"))
		}
		operation.Cls.Overrides = encryptedOverrides
		operation.Cls.Instance.Provisioned = true
		operation.Cls.Instance.ProvisioningTriggered = false
		// save the status
		operation, retry := s.operationManager.UpdateOperation(operation)
		if retry > 0 {
			log.Errorf("unable to update operation")
			return operation, time.Second, nil
		}
	} else {
		// get the credentials from encrypted string in operation.Cls.Instance.
		// clsOverrides, err = decryptOverrides(s.secretKey, operation.Cls.Overrides)
		_, err = decryptOverrides(s.secretKey, operation.Cls.Overrides)

		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("decryptOverrides() call failed"))
		}
	}

	// append overrides
	// operation.InputCreator.AppendOverrides(components.Eventing, getEventingOverrides(clsOverrides))

	return operation, 0, nil
}

func (s *ClsBindStep) handleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg)
}

// func getClsOverrides
