package provisioning

import (
	"fmt"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"

	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

// ServiceManagerOfferingStep checks if the ServiceManager has the expected offering and
// stores IDs of the offering, the broker and the plan
type ServiceManagerOfferingStep struct {
	stepName     string
	offeringName string
	planName     string

	operationManager *process.ProvisionOperationManager
	extractor        func(po *internal.ProvisioningOperation) *internal.ServiceManagerInstanceInfo
}

func NewServiceManagerOfferingStep(stepName, offeringName, planName string,
	extractor func(po *internal.ProvisioningOperation,
	) *internal.ServiceManagerInstanceInfo, repo storage.Operations) *ServiceManagerOfferingStep {
	return &ServiceManagerOfferingStep{
		operationManager: process.NewProvisionOperationManager(repo),
		extractor:        extractor,

		stepName:     stepName,
		planName:     planName,
		offeringName: offeringName,
	}
}

func (s *ServiceManagerOfferingStep) Name() string {
	return s.stepName
}

func (s *ServiceManagerOfferingStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	info := s.extractor(&operation)
	if info.ServiceID != "" && info.PlanID != "" {
		return operation, 0, nil
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, "unable to create Service Manager client", log)
	}

	// try to find the offering
	offerings, err := smCli.ListOfferingsByName(s.offeringName)
	if err != nil {
		return s.handleError(operation, err, "unable to get Service Manager offerings", log)
	}
	if len(offerings.ServiceOfferings) != 1 {
		return s.operationManager.OperationFailed(operation,
			fmt.Sprintf("expected one %s Service Manager offering, but found %d", s.offeringName, len(offerings.ServiceOfferings)))
	}
	info.ServiceID = offerings.ServiceOfferings[0].CatalogID
	info.BrokerID = offerings.ServiceOfferings[0].BrokerID
	log.Infof("Found offering: catalogID=%s brokerID=%s", info.ServiceID, info.BrokerID)

	// try to find the plan
	plans, err := smCli.ListPlansByName(s.planName, offerings.ServiceOfferings[0].ID)
	if err != nil {
		return s.handleError(operation, err, "unable to get Service Manager plan", log)
	}
	if len(plans.ServicePlans) != 1 {
		return s.operationManager.OperationFailed(operation,
			fmt.Sprintf("expected one %s Service Manager plan, but found %d", s.offeringName, len(offerings.ServiceOfferings)))
	}
	info.PlanID = plans.ServicePlans[0].CatalogID
	log.Infof("Found plan: catalogID=%s", info.PlanID)

	op, retry := s.operationManager.UpdateOperation(operation)
	if retry > 0 {
		log.Errorf("unable to update the operation")
		return op, retry, nil
	}
	return op, 0, nil
}

func (s *ServiceManagerOfferingStep) handleError(operation internal.ProvisioningOperation, err error, msg string, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	switch {
	case kebError.IsTemporaryError(err):
		return s.operationManager.RetryOperation(operation, msg, 10*time.Second, time.Minute*30, log)
	default:
		return s.operationManager.OperationFailed(operation, msg)
	}
}
