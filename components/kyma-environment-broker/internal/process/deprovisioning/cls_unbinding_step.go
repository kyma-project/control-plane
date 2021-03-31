package deprovisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type ClsUnbindStep struct {
	operationManager *process.DeprovisionOperationManager
	config           *cls.Config
}

func NewClsUnbindStep(config *cls.Config, os storage.Operations) *ClsUnbindStep {
	return &ClsUnbindStep{
		operationManager: process.NewDeprovisionOperationManager(os),
		config:           config,
	}
}

var _ Step = (*ClsUnbindStep)(nil)

func (s *ClsUnbindStep) Name() string {
	return "CLS_Unbind"
}

func (s *ClsUnbindStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if operation.Cls.Overrides == "" {
		log.Info("Cls Unbind step skipped, instance not bound")
		return operation, 0, nil
	}

	smCredentials, err := cls.FindCredentials(s.config.ServiceManager, operation.Cls.Region)
	if err != nil {
		failureReason := fmt.Sprintf("Unable to find credentials for cls service manager in region %s: %s", operation.Cls.Region, err)
		log.Error(failureReason)
		return s.operationManager.OperationFailed(operation, failureReason, log)
	}
	smCli := operation.SMClientFactory.ForCredentials(smCredentials)

	// Unbind
	log.Infof("Unbinding for CLS instance: %s started; binding: %s", operation.Cls.Instance.InstanceID, operation.Cls.BindingID)
	_, err = smCli.Unbind(operation.Cls.Instance.InstanceKey(), operation.Cls.BindingID, true)
	if err != nil {
		failureReason := "Unable to delete CLS Binding"
		log.Errorf("%s: %v", failureReason, err)
		if kebError.IsTemporaryError(err) {
			return s.operationManager.RetryOperation(operation, failureReason, 10*time.Second, time.Minute*30, log)
		}
		return s.operationManager.OperationFailed(operation, failureReason, log)
	}
	log.Infof("Unbinding for CLS instance: %s finished", operation.Cls.Instance.InstanceID)

	updatedOperation, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.DeprovisioningOperation) {
		operation.Cls.BindingID = ""
		operation.Cls.Overrides = ""
	}, log)
	return updatedOperation, retry, nil
}

func (s *ClsUnbindStep) handleError(operation internal.DeprovisioningOperation, err error, log logrus.FieldLogger,
	msg string) (internal.DeprovisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg, log)
}
