package deprovisioning

import (
	"fmt"
	"time"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type XSUAAUnbindStep struct {
	operationManager *process.DeprovisionOperationManager
}

var _ Step = (*XSUAAUnbindStep)(nil)

func NewXSUAAUnbindStep(os storage.Operations) *XSUAAUnbindStep {
	return &XSUAAUnbindStep{
		operationManager: process.NewDeprovisionOperationManager(os),
	}
}

func (s *XSUAAUnbindStep) Name() string {
	return "XSUAA_Unbind"
}

func (s *XSUAAUnbindStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (
	internal.DeprovisioningOperation, time.Duration, error) {
	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, "unable to create Service Manager client", log)
	}
	if operation.XSUAA.BindingID == "" {
		return operation, 0, nil
	}
	log.Infof("Triggering unbinding")
	_, err = smCli.Unbind(operation.XSUAA.Instance.InstanceKey(), operation.XSUAA.BindingID, false)
	if err != nil {
		return s.handleError(operation, err, fmt.Sprintf("unable to unbind, bindingId=%s", operation.XSUAA.BindingID), log)
	}
	operation.XSUAA.BindingID = ""
	return s.operationManager.UpdateOperation(operation)
}

func (s *XSUAAUnbindStep) handleError(operation internal.DeprovisioningOperation, err error, msg string, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	switch {
	case kebError.IsTemporaryError(err):
		return s.operationManager.RetryOperation(operation, msg, 10*time.Second, time.Minute*30, log)
	default:
		return s.operationManager.OperationFailed(operation, msg)
	}
}
