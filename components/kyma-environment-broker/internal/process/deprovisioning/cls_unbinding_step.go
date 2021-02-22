package deprovisioning

import (
	"fmt"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"time"

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
		config: config,
	}
}

var _ Step = (*ClsUnbindStep)(nil)

func (s *ClsUnbindStep) Name() string {
	return "Cls_Unbind"
}

func (s *ClsUnbindStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if operation.Cls.BindingID == "" {
		log.Infof("Cls Unbind step skipped, instance not bound")
		return operation, 0, nil
	}

	skrRegion := operation.ProvisioningParameters.Parameters.Region
	smRegion, err := cls.DetermineServiceManagerRegion(skrRegion)
	smCredentials, err := cls.FindCredentials(s.config.ServiceManager, smRegion)
	smCli := operation.SMClientFactory.ForCredentials(smCredentials)

	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manage client"))
	}

	// Unbind
	log.Infof("unbinding for CLS instance: %s started; binding: %s", operation.Cls.Instance.InstanceID, operation.Cls.BindingID)
	_, err = smCli.Unbind(operation.Cls.Instance.InstanceKey(), operation.Cls.BindingID, true)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to unbind, bindingId=%s", operation.Cls.BindingID))
	}
	log.Infof("unbinding for CLS instance: %s finished", operation.Cls.Instance.InstanceID)
	operation.Cls.BindingID = ""
	//operation.Cls.Binding.BindingID
	operation.Cls.Overrides = ""

	return s.operationManager.UpdateOperation(operation)
}

func (s *ClsUnbindStep) handleError(operation internal.DeprovisioningOperation, err error, log logrus.FieldLogger,
	msg string) (internal.DeprovisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg)
}
