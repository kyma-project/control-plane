package provisioning

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type SMBindingStep struct {
	operationManager *process.ProvisionOperationManager
}

func NewSMBindingStep(repo storage.Operations) *SMBindingStep {
	return &SMBindingStep{
		operationManager: process.NewProvisionOperationManager(repo),
	}
}

func (s *SMBindingStep) Name() string {
	return "SM_Binding"
}

func (s *SMBindingStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, "unable to create Service Manager client", log)
	}

	// check if the instance is provisioned
	if !operation.BTPOperator.Instance.ProvisioningTriggered {
		log.Warnf("Provisioning was not triggered")
		return operation, time.Second, nil
	}
	if !operation.BTPOperator.Instance.Provisioned {
		resp, err := smCli.LastInstanceOperation(operation.BTPOperator.Instance.InstanceKey(), "")
		if err != nil {
			return s.handleError(operation, err, "unable to create Service Manage client", log)
		}
		log.Infof("Provisioning sm (instanceID=%s) state: %s", resp.State)
		switch resp.State {
		case servicemanager.InProgress:
			return operation, 15 * time.Second, nil
		case servicemanager.Failed:
			return s.operationManager.OperationFailed(operation, fmt.Sprintf("sm provisioning failed: %s", resp.Description), log)
		}
	}

	// execute binding
	if operation.BTPOperator.BindingID == "" {
		operation, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
			operation.BTPOperator.BindingID = uuid.New().String()
		}, log)
		if retry > 0 {
			log.Errorf("unable to update operation")
			return operation, time.Second, nil
		}
	}

	resp, err := smCli.Bind(operation.BTPOperator.Instance.InstanceKey(), operation.BTPOperator.BindingID, nil, false)
	if err != nil {
		return s.handleError(operation, err, "unable to execute binding", log)
	}

	log.Infof("Got binding keys:")
	for k, _ := range resp.Credentials {
		log.Info(k)
	}

	return operation, 0, nil
}

func (s *SMBindingStep) handleError(operation internal.ProvisioningOperation, err error, msg string, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	switch {
	case kebError.IsTemporaryError(err):
		return s.operationManager.RetryOperation(operation, msg, 10*time.Second, time.Minute*30, log)
	default:
		return s.operationManager.OperationFailed(operation, msg, log)
	}
}
