package deprovisioning

import (
	"time"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type XSUAADeprovisionStep struct {
	operationManager *process.DeprovisionOperationManager
}

func NewXSUAADeprovisionStep(os storage.Operations) *XSUAADeprovisionStep {
	return &XSUAADeprovisionStep{
		operationManager: process.NewDeprovisionOperationManager(os),
	}
}

var _ Step = (*XSUAADeprovisionStep)(nil)

func (s *XSUAADeprovisionStep) Name() string {
	return "XSUAA_Deprovision"
}

func (s *XSUAADeprovisionStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (
	internal.DeprovisioningOperation, time.Duration, error) {
	smcli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, "unable to create Service Manager client", log)
	}
	if operation.XSUAA.Instance.InstanceID == "" {
		return operation, 0, nil
	}
	log.Infof("Triggering deprovision")
	_, err = smcli.Deprovision(operation.XSUAA.Instance.InstanceKey(), false)
	if err != nil {
		return s.handleError(operation, err, "unable to deprovision", log)
	}
	updatedOperation, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.DeprovisioningOperation) {
		operation.XSUAA.Instance.InstanceID = ""
	}, log)
	return updatedOperation, retry, nil
}

func (s *XSUAADeprovisionStep) handleError(operation internal.DeprovisioningOperation, err error, msg string, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	switch {
	case kebError.IsTemporaryError(err):
		return s.operationManager.RetryOperation(operation, msg, 10*time.Second, time.Minute*30, log)
	default:
		return s.operationManager.OperationFailed(operation, msg, log)
	}
}
