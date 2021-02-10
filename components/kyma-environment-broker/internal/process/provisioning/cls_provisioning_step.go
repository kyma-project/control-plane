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
	ProvisionIfNoneExists(om *process.ProvisionOperationManager, smCli servicemanager.Client, op internal.ProvisioningOperation, globalAccountID string) (internal.ProvisioningOperation, error)
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
	if operation.Cls.Instance.ProvisioningTriggered {
		return operation, 0, nil
	}

	skrRegion := operation.ProvisioningParameters.Parameters.Region
	smCli, err := cls.ServiceManagerClient(operation.SMClientFactory, s.config.ServiceManager, skrRegion)

	globalAccountID := operation.ProvisioningParameters.ErsContext.GlobalAccountID
	op, err := s.instanceProvider.ProvisionIfNoneExists(s.operationManager, smCli, operation, globalAccountID)
	if err != nil {
		failureReason := fmt.Sprintf("Unable to create instance for GlobalAccountID: %s", globalAccountID)
		log.Errorf("%s: %s", failureReason, err)
		return s.operationManager.OperationFailed(operation, failureReason)
	}

	_, repeat := s.operationManager.UpdateOperation(op)
	if repeat != 0 {
		log.Errorf("Unable to update operation: %s", err)
		return operation, time.Second, nil
	}

	return op, 0, nil
}
