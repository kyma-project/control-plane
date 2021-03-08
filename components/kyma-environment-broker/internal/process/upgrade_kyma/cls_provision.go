package upgrade_kyma

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type clsUpgradeProvisionStep struct {
	config           *cls.Config
	instanceProvider provisioning.ClsProvisioner
	operationManager *process.UpgradeKymaOperationManager
}

func NewClsUpgradeProvisionStep(config *cls.Config, ip provisioning.ClsProvisioner, repo storage.Operations) *clsUpgradeProvisionStep {
	return &clsUpgradeProvisionStep{
		config:           config,
		operationManager: process.NewUpgradeKymaOperationManager(repo),
		instanceProvider: ip,
	}
}

func (s *clsUpgradeProvisionStep) Name() string {
	return "CLS_UpgradeProvision"
}

func (s *clsUpgradeProvisionStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if operation.Cls.Instance.InstanceID != "" {
		log.Infof("CLS instance already exists")
		return operation, 0, nil
	}

	globalAccountID := operation.ProvisioningParameters.ErsContext.GlobalAccountID

	skrRegion := operation.ProvisioningParameters.Parameters.Region
	smRegion, err := cls.DetermineServiceManagerRegion(skrRegion)
	if err != nil {
		failureReason := fmt.Sprintf("Unable to determine cls service manager region %v: %s", skrRegion, err)
		log.Error(failureReason)
		return s.operationManager.OperationFailed(operation, failureReason)
	}

	smCredentials, err := cls.FindCredentials(s.config.ServiceManager, smRegion)
	if err != nil {
		failureReason := fmt.Sprintf("Unable to find credentials for cls service manager in region %s: %s", operation.Cls.Region, err)
		log.Error(failureReason)
		return s.operationManager.OperationFailed(operation, failureReason)
	}

	log.Infof("Starting provisioning a cls instance for global account %s", globalAccountID)

	smClient := operation.SMClientFactory.ForCredentials(smCredentials)
	skrInstanceID := operation.InstanceID
	result, err := s.instanceProvider.Provision(smClient, &cls.ProvisionRequest{
		GlobalAccountID: globalAccountID,
		Region:          smRegion,
		SKRInstanceID:   skrInstanceID,
		Instance:        operation.Cls.Instance.InstanceKey(),
	})
	if err != nil {
		failureReason := fmt.Sprintf("Unable to provision a cls instance for global account %s: %s", globalAccountID, err)
		log.Error(failureReason)
		return s.operationManager.OperationFailed(operation, failureReason)
	}

	operation.Cls.Region = result.Region
	operation.Cls.Instance.InstanceID = result.InstanceID
	operation.Cls.Instance.ProvisioningTriggered = result.ProvisioningTriggered

	log.Infof("Finished provisioning a cls instance for global account %s", globalAccountID)

	_, repeat := s.operationManager.UpdateOperation(operation)
	if repeat != 0 {
		log.Errorf("Unable to update operation: %s", err)
		return operation, time.Second, nil
	}

	return operation, 0, nil
}
