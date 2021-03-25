package provisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

//go:generate mockery --name=ClsProvisioner --output=automock --outpkg=automock --case=underscore
type ClsProvisioner interface {
	Provision(smClient servicemanager.Client, request *cls.ProvisionRequest, log logrus.FieldLogger) (*cls.ProvisionResult, error)
}

type clsProvisionStep struct {
	config           *cls.Config
	provisioner      ClsProvisioner
	operationManager *process.ProvisionOperationManager
}

func NewClsProvisionStep(config *cls.Config, provisioner ClsProvisioner, repo storage.Operations) *clsProvisionStep {
	return &clsProvisionStep{
		config:           config,
		provisioner:      provisioner,
		operationManager: process.NewProvisionOperationManager(repo),
	}
}

func (s *clsProvisionStep) Name() string {
	return "CLS_Provision"
}

func (s *clsProvisionStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if operation.Cls.Instance.InstanceID != "" {
		log.Infof("CLS instance already exists")
		return operation, 0, nil
	}

	globalAccountID := operation.ProvisioningParameters.ErsContext.GlobalAccountID

	skrRegion := operation.ProvisioningParameters.Parameters.Region
	smRegion := cls.DetermineServiceManagerRegion(skrRegion, log)
	smCredentials, err := cls.FindCredentials(s.config.ServiceManager, smRegion)
	if err != nil {
		failureReason := fmt.Sprintf("Unable to find credentials for CLS Service Manager in region %s", operation.Cls.Region)
		log.Errorf("%s: %v", failureReason, err)
		return s.operationManager.OperationFailed(operation, failureReason, log)
	}

	log.Infof("Starting provisioning a CLS instance for global account %s", globalAccountID)

	smClient := operation.SMClientFactory.ForCredentials(smCredentials)
	skrInstanceID := operation.InstanceID
	result, err := s.provisioner.Provision(smClient, &cls.ProvisionRequest{
		GlobalAccountID: globalAccountID,
		Region:          smRegion,
		SKRInstanceID:   skrInstanceID,
		Instance:        operation.Cls.Instance.InstanceKey(),
	}, log)
	if err != nil {
		failureReason := fmt.Sprintf("Unable to provision a CLS instance for global account %s", globalAccountID)
		log.Errorf("%s: %v", failureReason, err)
		if kebError.IsTemporaryError(err) {
			return s.operationManager.RetryOperation(operation, failureReason, 10*time.Second, time.Minute*30, log)
		}
		return s.operationManager.OperationFailed(operation, failureReason, log)
	}

	log.Infof("Finished provisioning a CLS instance for global account %s", globalAccountID)

	op, repeat := s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
		operation.Cls.Region = result.Region
		operation.Cls.Instance.InstanceID = result.InstanceID
		operation.Cls.Instance.ProvisioningTriggered = true
	}, log)
	if repeat != 0 {
		log.Errorf("Unable to update operation")
		return operation, time.Second, nil
	}

	return op, 0, nil
}
