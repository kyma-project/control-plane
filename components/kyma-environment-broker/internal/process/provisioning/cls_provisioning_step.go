package provisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type ClsInstanceProvider interface {
	ProvideClsInstanceID(om *process.ProvisionOperationManager, smCli servicemanager.Client, op internal.ProvisioningOperation, globalAccountID string) (internal.ProvisioningOperation, error)
}

type clsProvisioningStep struct {
	config           *cls.Config
	instanceProvider ClsInstanceProvider
	operationManager *process.ProvisionOperationManager
}

func NewClsProvisioningStep(config *cls.Config, ip ClsInstanceProvider, repo storage.Operations) *clsProvisioningStep {
	return &clsProvisioningStep{
		config:           config,
		operationManager: process.NewProvisionOperationManager(repo),
		instanceProvider: ip,
	}
}

func (s *clsProvisioningStep) Name() string {
	return "CLS_Provision"
}

func (s *clsProvisioningStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	// TODO: Fetch if there is already a CLS assigned to the GA. If so then we dont need to provision a new one.
	if operation.Cls.Instance.InstanceID != "" {
		return operation, 0, nil
	}

	smCli, err := cls.ServiceManagerClient(s.config.ServiceManager, &operation)

	op, err := s.instanceProvider.ProvideClsInstanceID(s.operationManager, smCli, operation, operation.ProvisioningParameters.ErsContext.GlobalAccountID)
	if err != nil {
		return s.handleError(
			operation,
			err,
			log,
			fmt.Sprintf("Unable to get tenant for GlobalaccountID %s", operation.ProvisioningParameters.ErsContext.GlobalAccountID))
	}

	op.Cls.Instance.ProvisioningTriggered = true

	_, repeat := s.operationManager.UpdateOperation(op)
	if repeat != 0 {
		s.handleError(op, err, log, fmt.Sprintf("cannot save LMS tenant ID"))
		return operation, time.Second, nil
	}

	return op, 0, nil
}

func (s *clsProvisioningStep) handleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg)
}
