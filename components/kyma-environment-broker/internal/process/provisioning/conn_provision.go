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
	ConnOfferingName = "connectivity"
	ConnPlanName     = "lite"
)

type ConnProvisionStep struct {
	operationManager *process.ProvisionOperationManager
}

func NewConnProvisionStep(os storage.Operations) *ConnProvisionStep {
	return &ConnProvisionStep{
		operationManager: process.NewProvisionOperationManager(os),
	}
}

var _ Step = (*ConnProvisionStep)(nil)

func (s *ConnProvisionStep) Name() string {
	return "CONN_Provision"
}

func (s *ConnProvisionStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if operation.Conn.Instance.ProvisioningTriggered {
		log.Infof("Conn Provisioning step was already triggered")
		return operation, 0, nil
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manage client"))
	}

	if operation.Conn.Instance.InstanceID == "" {
		op, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
			operation.Conn.Instance.InstanceID = uuid.New().String()
		}, log)
		if retry > 0 {
			log.Errorf("unable to update operation")
			return operation, time.Second, nil
		}
		operation = op
	}

	// provision
	operation, _, err = s.provision(smCli, operation, log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("provision()  call failed"))
	}
	// save the status
	operation, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
		operation.Conn.Instance.ProvisioningTriggered = true
	}, log)
	if retry > 0 {
		log.Errorf("unable to update operation")
		return operation, time.Second, nil
	}

	return operation, 0, nil
}

func (s *ConnProvisionStep) provision(smCli servicemanager.Client, operation internal.ProvisioningOperation,
	log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {

	input := getEventingProvisioningData(operation.Conn)
	resp, err := smCli.Provision(operation.Conn.Instance.BrokerID, *input, true)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("Provision() call failed for brokerID: %s; input: %#v", operation.Conn.Instance.BrokerID, input))
	}
	log.Debugf("response from Conn provisioning call: %#v", resp)

	return operation, 0, nil
}

func (s *ConnProvisionStep) handleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg, log)
}

func getEventingProvisioningData(connInstanceData internal.ConnData) *servicemanager.ProvisioningInput {
	var input servicemanager.ProvisioningInput

	input.ID = connInstanceData.Instance.InstanceID
	input.ServiceID = connInstanceData.Instance.ServiceID
	input.PlanID = connInstanceData.Instance.PlanID
	input.SpaceGUID = uuid.New().String()
	input.OrganizationGUID = uuid.New().String()

	input.Context = map[string]interface{}{
		"platform": "kubernetes",
	}
	input.Parameters = map[string]interface{}{
		// TODO: Fill
	}

	return &input
}
