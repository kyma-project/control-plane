package deprovisioning

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

//go:generate mockery --name=ClsDeprovisioner --output=automock --outpkg=automock --case=underscore
type ClsDeprovisioner interface {
	Deprovision(smClient servicemanager.Client, request *cls.DeprovisionRequest) error
}

type ClsDeprovisionStep struct {
	config           *cls.Config
	operationManager *process.DeprovisionOperationManager
	deprovisioner    ClsDeprovisioner
}

func NewClsDeprovisionStep(config *cls.Config, os storage.Operations, deprovisioner ClsDeprovisioner) *ClsDeprovisionStep {
	return &ClsDeprovisionStep{
		config:           config,
		operationManager: process.NewDeprovisionOperationManager(os),
		deprovisioner:    deprovisioner,
	}
}

func (s *ClsDeprovisionStep) Name() string {
	return "CLS_Deprovision"
}

func (s *ClsDeprovisionStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	globalAccountID := operation.ProvisioningParameters.ErsContext.GlobalAccountID
	skrInstanceID := operation.InstanceID

	if !operation.Cls.Instance.Provisioned {
		log.Warnf("Unable to deprovision a cls instance for global account %s since it is not provisioned", globalAccountID)
		return operation, 0, nil
	}

	smCredentials, err := cls.FindCredentials(s.config.ServiceManager, operation.Cls.Region)
	if err != nil {
		failureReason := fmt.Sprintf("Unable to find credentials for cls service manager in region %s: %s", operation.Cls.Region, err)
		log.Error(failureReason)
		return s.operationManager.OperationFailed(operation, failureReason, log)
	}

	smClient := operation.SMClientFactory.ForCredentials(smCredentials)

	request := &cls.DeprovisionRequest{
		SKRInstanceID: skrInstanceID,
		Instance:      operation.Cls.Instance.InstanceKey(),
	}

	if !operation.Cls.Instance.DeprovisioningTriggered {
		if err := s.deprovisioner.Deprovision(smClient, request); err != nil {
			failureReason := fmt.Sprintf("Unable to deprovision a cls instance %s: %s", operation.Cls.Instance.InstanceID, err)
			log.Error(failureReason)
			return s.operationManager.RetryOperation(operation, failureReason, 1*time.Minute, 5*time.Minute, log)
		}
		operation, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.DeprovisioningOperation) {
			operation.Cls.Instance.DeprovisioningTriggered = true
		}, log)
		return operation, retry, nil
	}

	return s.checkDeprovisioningStatus(operation, log, smClient)
}

func (s *ClsDeprovisionStep) checkDeprovisioningStatus(operation internal.DeprovisioningOperation, log logrus.FieldLogger, smClient servicemanager.Client) (internal.DeprovisioningOperation, time.Duration, error) {
	instanceID := operation.Cls.Instance.InstanceID

	resp, err := smClient.LastInstanceOperation(operation.Cls.Instance.InstanceKey(), "")
	if err != nil {
		failureReason := fmt.Sprintf("Unable to poll the status of a cls instance %s: %s", instanceID, err)
		log.Error(failureReason)
		return s.operationManager.RetryOperation(operation, failureReason, 1*time.Minute, 5*time.Minute, log)
	}

	switch resp.State {
	case servicemanager.InProgress:
		return operation, 30 * time.Second, nil
	case servicemanager.Failed:
		failureReason := fmt.Sprintf("Deprovisioning of a cls instance %s failed", instanceID)
		log.Error(failureReason)
		return s.operationManager.OperationFailed(operation, failureReason, log)
	default:
		log.Debugf("Finished deprovisioning a cls instance %s", instanceID)
		updatedOperation, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.DeprovisioningOperation) {
			operation.Cls.Instance.InstanceID = ""
			operation.Cls.Instance.Provisioned = false
		}, log)
		return updatedOperation, retry, nil
	}
}
