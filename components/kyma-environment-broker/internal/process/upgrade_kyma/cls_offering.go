package upgrade_kyma

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
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
	info := &operation.Cls.Instance

	if info.ServiceID != "" && info.PlanID != "" {
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

	// try to find the offering
	offerings, err := smClient.ListOfferingsByName(provisioning.ClsOfferingName)
	if err != nil {
		return s.handleError(operation, err, "unable to get Service Manager offerings", log)
	}
	if len(offerings.ServiceOfferings) != 1 {
		return s.operationManager.OperationFailed(operation,
			fmt.Sprintf("expected one %s Service Manager offering, but found %d", provisioning.ClsOfferingName, len(offerings.ServiceOfferings)), log)
	}
	info.ServiceID = offerings.ServiceOfferings[0].CatalogID
	info.BrokerID = offerings.ServiceOfferings[0].BrokerID
	log.Infof("Found offering: catalogID=%s brokerID=%s", info.ServiceID, info.BrokerID)

	// try to find the plan
	plans, err := smClient.ListPlansByName(provisioning.ClsPlanName, offerings.ServiceOfferings[0].ID)
	if err != nil {
		return s.handleError(operation, err, "unable to get Service Manager plan", log)
	}
	if len(plans.ServicePlans) != 1 {
		return s.operationManager.OperationFailed(operation,
			fmt.Sprintf("expected one %s Service Manager plan, but found %d", provisioning.ClsPlanName, len(offerings.ServiceOfferings)), log)
	}
	info.PlanID = plans.ServicePlans[0].CatalogID
	log.Infof("Found plan: catalogID=%s", info.PlanID)

	op, retry := s.operationManager.SimpleUpdateOperation(operation)
	if retry > 0 {
		log.Errorf("unable to update the operation")
		return op, retry, nil
	}
	return op, 0, nil
}

func (s *ClsUpgradeOfferingStep) handleError(operation internal.UpgradeKymaOperation, err error, msg string, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	switch {
	case kebError.IsTemporaryError(err):
		return s.operationManager.RetryOperation(operation, msg, 10*time.Second, time.Minute*30, log)
	default:
		return s.operationManager.OperationFailed(operation, msg, log)
	}
}
