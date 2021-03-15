package upgrade_kyma

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"

	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type ClsUpgradeOfferingStep struct {
	config           *cls.Config
	operationManager *process.UpgradeKymaOperationManager
}

func NewClsUpgradeOfferingStep(config *cls.Config, repo storage.Operations) *ClsUpgradeOfferingStep {
	return &ClsUpgradeOfferingStep{
		config:           config,
		operationManager: process.NewUpgradeKymaOperationManager(repo),
	}
}

func (s *ClsUpgradeOfferingStep) Name() string {
	return "CLS_UpgradeOffering"
}

func (s *ClsUpgradeOfferingStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if operation.Cls.Instance.ServiceID != "" && operation.Cls.Instance.PlanID != "" {
		return operation, 0, nil
	}

	skrRegion := operation.ProvisioningParameters.Parameters.Region
	smRegion, err := cls.DetermineServiceManagerRegion(skrRegion)
	if err != nil {
		return s.handleError(operation, err, err.Error(), log)
	}

	smCredentials, err := cls.FindCredentials(s.config.ServiceManager, smRegion)
	if err != nil {
		return s.handleError(operation, err, err.Error(), log)
	}

	smClient := operation.SMClientFactory.ForCredentials(smCredentials)

	meta, err := servicemanager.GenerateMetadata(smClient, provisioning.ClsOfferingName, provisioning.ClsPlanName)
	if meta.ServiceID != "" && meta.BrokerID != "" {
		log.Debugf("Found offering with catalog ID %s and broker ID %s", meta.ServiceID, meta.BrokerID)
	}
	if err != nil {
		return s.handleError(operation, err, err.Error(), log)
	}

	log.Debugf("Found plan with catalog ID %s", meta.PlanID)

	op, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.UpgradeKymaOperation) {
		operation.Cls.Instance.ServiceID = meta.ServiceID
		operation.Cls.Instance.BrokerID = meta.BrokerID
		operation.Cls.Instance.PlanID = meta.PlanID
	}, log)
	if retry > 0 {
		log.Errorf("Unable to update the operation")
		return op, retry, nil
	}
	return op, 0, nil
}

func (s *ClsUpgradeOfferingStep) handleError(operation internal.UpgradeKymaOperation, err error, msg string, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	log.Errorf("%s: %v", msg, err)
	switch {
	case kebError.IsTemporaryError(err):
		return s.operationManager.RetryOperation(operation, msg, 10*time.Second, time.Minute*30, log)
	default:
		return s.operationManager.OperationFailed(operation, msg, log)
	}
}
