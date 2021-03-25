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
	Deprovision(smClient servicemanager.Client, request *cls.DeprovisionRequest, log logrus.FieldLogger) (*cls.DeprovisionResult, error)
}

type ClsDeprovisionStep struct {
	config           *cls.Config
	deprovisioner    ClsDeprovisioner
	operationManager *process.DeprovisionOperationManager
}

func NewClsDeprovisionStep(config *cls.Config, deprovisioner ClsDeprovisioner, os storage.Operations) *ClsDeprovisionStep {
	return &ClsDeprovisionStep{
		config:           config,
		deprovisioner:    deprovisioner,
		operationManager: process.NewDeprovisionOperationManager(os),
	}
}

func (s *ClsDeprovisionStep) Name() string {
	return "CLS_Deprovision"
}

func (s *ClsDeprovisionStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	globalAccountID := operation.ProvisioningParameters.ErsContext.GlobalAccountID
	skrInstanceID := operation.InstanceID

	if operation.Cls.Instance.InstanceID == "" {
		log.Warnf("Unable to deprovision a CLS instance for global account %s since it is not provisioned", globalAccountID)
		return operation, 0, nil
	}

	smCredentials, err := cls.FindCredentials(s.config.ServiceManager, operation.Cls.Region)
	if err != nil {
		failureReason := fmt.Sprintf("Unable to find credentials for CLS service manager in region %s: %s", operation.Cls.Region, err)
		log.Error(failureReason)
		return s.operationManager.OperationFailed(operation, failureReason, log)
	}

	smClient := operation.SMClientFactory.ForCredentials(smCredentials)

	request := &cls.DeprovisionRequest{
		SKRInstanceID: skrInstanceID,
		Instance:      operation.Cls.Instance.InstanceKey(),
	}

	if !operation.Cls.Instance.DeprovisioningTriggered {
		result, err := s.deprovisioner.Deprovision(smClient, request, log)
		if err != nil {
			failureReason := fmt.Sprintf("Unable to deprovision a CLS instance %s: %s", operation.Cls.Instance.InstanceID, err)
			log.Error(failureReason)
			return s.operationManager.RetryOperation(operation, failureReason, 1*time.Minute, 5*time.Minute, log)
		}
		updatedOperation, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.DeprovisioningOperation) {
			operation.Cls.Instance.DeprovisioningTriggered = true
		}, log)
		if retry > 0 {
			log.Errorf("Unable to update operation")
			return operation, retry, nil
		}

		if result.IsLastReference {
			return updatedOperation, 10 * time.Second, nil
		}

		return updatedOperation, 0, nil
	}

	return s.checkDeprovisioningStatus(operation, log, smClient)
}

func (s *ClsDeprovisionStep) checkDeprovisioningStatus(operation internal.DeprovisioningOperation, log logrus.FieldLogger, smClient servicemanager.Client) (internal.DeprovisioningOperation, time.Duration, error) {
	instanceID := operation.Cls.Instance.InstanceID

	resp, err := smClient.LastInstanceOperation(operation.Cls.Instance.InstanceKey(), "")
	if err != nil {
		failureReason := fmt.Sprintf("Unable to poll the status of a CLS instance %s: %s", instanceID, err)
		log.Error(failureReason)
		return s.operationManager.RetryOperation(operation, failureReason, 1*time.Minute, 5*time.Minute, log)
	}

	switch resp.State {
	case servicemanager.InProgress:
		log.Infof("Deprovisioning a CLS instance %s is in progress. Retrying", instanceID)
		return operation, 10 * time.Second, nil
	case servicemanager.Failed:
		failureReason := fmt.Sprintf("Deprovisioning of a CLS instance %s failed", instanceID)
		log.Error(failureReason)
		return s.operationManager.OperationFailed(operation, failureReason, log)
	case servicemanager.Succeeded:
		log.Infof("Finished deprovisioning a CLS instance %s", instanceID)
		updatedOperation, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.DeprovisioningOperation) {
			operation.Cls.Instance.InstanceID = ""
			operation.Cls.Instance.Provisioned = false
		}, log)
		return updatedOperation, retry, nil
	}

	return operation, 0, nil
}
