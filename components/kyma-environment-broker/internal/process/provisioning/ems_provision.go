package provisioning

import (
	"fmt"
	"time"

	"github.com/Peripli/service-manager-cli/pkg/types"
	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

const (
	EmsOfferingName = "enterprise-messaging"
	EmsPlanName     = "default"
)

type EmsProvisionStep struct {
	operationManager *process.ProvisionOperationManager
}

func NewEmsProvisionStep(os storage.Operations) *EmsProvisionStep {
	return &EmsProvisionStep{
		operationManager: process.NewProvisionOperationManager(os),
	}
}

var _ Step = (*EmsProvisionStep)(nil)

func (s *EmsProvisionStep) Name() string {
	return "EMS_Provision"
}

func (s *EmsProvisionStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if operation.Ems.Instance.InstanceID != "" {
		return operation, 0, nil
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, "unable to create Service Manage client")
	}
	// get offering
	offs, err := smCli.ListOfferingsByName(EmsOfferingName)
	if err != nil {
		return s.handleError(operation, err, log, "ListOfferingsByName() failed for:"+EmsOfferingName)
	}
	if offs.IsEmpty() {
		return s.handleError(operation, fmt.Errorf("no offering for: %s", EmsOfferingName), log, "")
	}
	offering := offs.ServiceOfferings[0]
	// get plan
	plans, err := smCli.ListPlansByName(EmsPlanName, offering.ID)
	for _, plan := range plans.ServicePlans {
		log.Infof("plan: %s %s %s", plan.Name, plan.Description, plan.Message())
	}
	if plans.IsEmpty() {
		return s.handleError(operation, fmt.Errorf("no plans for: %s", EmsPlanName), log, "")
	}
	plan := plans.ServicePlans[0]
	log.Print(">>> plan: %#v\n", plan)

	// provision
	operation, _, err = s.provision(smCli, operation, log, offering, plan)
	if err != nil {
		return s.handleError(operation, err, log, "EMS provision operation failed")
	}

	operation, retry := s.operationManager.UpdateOperation(operation)
	if retry > 0 {
		log.Errorf("unable to update operation")
		return operation, time.Second, nil
	}

	return operation, 0, nil
}

func (s *EmsProvisionStep) provision(smCli servicemanager.Client, operation internal.ProvisioningOperation, log logrus.FieldLogger,
	offering types.ServiceOffering, plan types.ServicePlan) (internal.ProvisioningOperation, time.Duration, error) {

	var input servicemanager.ProvisioningInput
	input.ID = uuid.New().String() // is this OK?
	input.ServiceID = offering.CatalogID
	input.PlanID = plan.CatalogID
	input.SpaceGUID = uuid.New().String()
	input.OrganizationGUID = uuid.New().String()
	input.Context = map[string]interface{}{
		"platform": "kubernetes",
	}
	input.Parameters = map[string]interface{}{
		"options": map[string]string{
			"management":    "true",
			"messagingrest": "true",
		},
		"rules": map[string]interface{}{
			"topicRules": map[string]interface{}{
				"publishFilter": []string{
					"${namespace}/*",
				},
				"subscribeFilter": []string{
					"${namespace}/*",
				},
			},
			"queueRules": map[string]interface{}{
				"publishFilter": []string{
					"${namespace}/*",
				},
				"subscribeFilter": []string{
					"${namespace}/*",
				},
			},
		},
		"version":   "1.1.0",
		"emname":    uuid.New().String(),
		"namespace": "default/sap.kyma/" + uuid.New().String(),
	}

	resp, err := smCli.Provision(offering.BrokerID, input, true)
	log.Printf(">>> Response from Provision() err: %s; resp: %#v\n", err, resp)

	operation.Ems.Instance.BrokerID = offering.BrokerID
	operation.Ems.Instance.InstanceID = input.ID
	operation.Ems.Instance.ServiceID = offering.CatalogID
	operation.Ems.Instance.PlanID = input.PlanID

	return operation, 0, nil
}

func (s *EmsProvisionStep) handleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg)
}
