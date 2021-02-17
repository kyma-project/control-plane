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
	var (
		globalAccountID = operation.ProvisioningParameters.ErsContext.GlobalAccountID
		skrInstanceID   = operation.InstanceID
	)

	if !operation.Cls.Instance.Provisioned {
		log.Infof("Unable to deprovision a cls instance since it is not provisioned")
		return operation, 0, nil
	}

	log.Infof("Starting deprovisioning a cls instance for global account %s", globalAccountID)

	smCredentials, err := cls.FindCredentials(s.config.ServiceManager, operation.Cls.Region)
	if err != nil {
		failureReason := fmt.Sprintf("Unable to find credentials for cls service manager in region %s: %s", operation.Cls.Region, err)
		log.Error(failureReason)
		return s.operationManager.OperationFailed(operation, failureReason)
	}

	smClient := operation.SMClientFactory.ForCredentials(smCredentials)

	request := &cls.DeprovisionRequest{
		GlobalAccountID: globalAccountID,
		SKRInstanceID:   skrInstanceID,
		Instance:        operation.Cls.Instance.InstanceKey(),
	}

	if err := s.deprovisioner.Deprovision(smClient, request); err != nil {
		failureReason := fmt.Sprintf("Unable to deprovision a cls instance for global account %s: %s", globalAccountID, err)
		log.Error(failureReason)
		return s.operationManager.RetryOperation(operation, failureReason, 1*time.Minute, 5*time.Minute, log)
	}

	log.Infof("Finished deprovisioning a cls instance for global account %s", globalAccountID)

	operation.Cls.Instance.InstanceID = ""
	operation.Cls.Instance.Provisioned = false

	return s.operationManager.UpdateOperation(operation)
}
