package provisioning

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"time"

	"github.com/sirupsen/logrus"
)

//go:generate mockery --name=ClsStatusChecker --output=automock --outpkg=automock --case=underscore
type ClsStatusChecker interface {
	CheckProvisionStatus(smClient servicemanager.Client, instanceKey servicemanager.InstanceKey, log logrus.FieldLogger) (cls.ProvisionStatus, error)
}

type ClsCheckStatusStep struct {
	config           *cls.Config
	operationManager *process.ProvisionOperationManager
	statusChecker    ClsStatusChecker
}

func NewClsCheckStatus(config *cls.Config, sc ClsStatusChecker, os storage.Operations) *ClsCheckStatusStep {
	return &ClsCheckStatusStep{
		config:           config,
		operationManager: process.NewProvisionOperationManager(os),
		statusChecker:    sc,
	}
}

var _ Step = (*ClsCheckStatusStep)(nil)

func (s *ClsCheckStatusStep) Name() string {
	return "CLS_Check_Instance_Status"
}

func (s *ClsCheckStatusStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if operation.Cls.Instance.InstanceID == "" {
		failureReason := "CLS provisioning step was not triggered"
		log.Error(failureReason)
		return s.operationManager.OperationFailed(operation, failureReason, log)
	}

	smCredentials, err := cls.FindCredentials(s.config.ServiceManager, operation.Cls.Region)
	if err != nil {
		failureReason := fmt.Sprintf("Unable to find credentials for CLS Service Manager in region %s", operation.Cls.Region)
		log.Errorf("%s: %v", failureReason, err)
		return s.operationManager.OperationFailed(operation, failureReason, log)
	}
	smCli := operation.SMClientFactory.ForCredentials(smCredentials)

	res, err := s.statusChecker.CheckProvisionStatus(smCli, operation.Cls.Instance.InstanceKey(), log)
	switch res {
	case cls.Failed:
		failureReason := fmt.Sprintf("Unable to check status for CLS instance: %s", operation.Cls.Instance.InstanceID)
		log.Errorf("%s: %v", failureReason, err)
		return s.operationManager.OperationFailed(operation, failureReason, log)
	case cls.Retry:
		failureReason := fmt.Sprintf("Got following temporary error while checking status for CLS instance: %s, retrying", operation.Cls.Instance.InstanceID)
		log.Errorf("%s: %v", failureReason, err)
		return operation, time.Minute, nil
	case cls.InProgress:
		log.Infof("CLS Instance: %s is still provisioning", operation.Cls.Instance.InstanceID)
		return operation, time.Minute, nil
	case cls.Succeeded:
		log.Infof("CLS Instance successfully provisioned")
	}

	op, retry := s.operationManager.UpdateOperation(operation, func(op *internal.ProvisioningOperation) {
		op.Cls.Instance.Provisioned = true
	}, log)
	if retry > 0 {
		log.Errorf("Unable to update operation")
		return op, time.Second, nil
	}

	return op, 0, nil
}
