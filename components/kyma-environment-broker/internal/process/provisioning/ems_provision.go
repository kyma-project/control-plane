package provisioning

import (
	"fmt"
	"time"

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
	if operation.Ems.Instance.ProvisioningTriggered {
		log.Infof("Ems Provisioning step was already triggered")
		return operation, 0, nil
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manage client"))
	}

	// provision
	operation, _, err = s.provision(smCli, operation, log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("provision()  call failed"))
	}
	// save the status
	operation.Ems.Instance.ProvisioningTriggered = true
	operation, retry := s.operationManager.UpdateOperation(operation)
	if retry > 0 {
		log.Errorf("unable to update operation")
		return operation, time.Second, nil
	}

	return operation, 0, nil
}

func (s *EmsProvisionStep) provision(smCli servicemanager.Client, operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {

	var input servicemanager.ProvisioningInput
	input.ID = uuid.New().String()
	input.ServiceID = operation.Ems.Instance.ServiceID
	input.PlanID = operation.Ems.Instance.PlanID
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

	resp, err := smCli.Provision(operation.Ems.Instance.BrokerID, input, true)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("Provision() call failed for brokerID: %s; input: %#v", operation.Ems.Instance.BrokerID, input))
	}
	log.Infof("response from EMS provisioning call: %#v", resp)

	operation.Ems.Instance.InstanceID = input.ID

	return operation, 0, nil
}

func (s *EmsProvisionStep) handleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg)
}
