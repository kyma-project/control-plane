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

type clsParameters struct {
	RetentionPeriod    int  `json:"retentionPeriod"`
	MaxDataInstances   int  `json:"maxDataInstances"`
	MaxIngestInstances int  `json:"maxIngestInstances"`
	EsAPIEnabled       bool `json:"esApiEnabled"`
}

type ClsProvisioningStep struct {
	operationManager *process.ProvisionOperationManager
}

func NewClsProvisioningStep(os storage.Operations) *ClsProvisioningStep {
	return &ClsProvisioningStep{
		operationManager: process.NewProvisionOperationManager(os),
	}
}

var _ Step = (*ClsProvisioningStep)(nil)

func (s *ClsProvisioningStep) Name() string {
	return "CLS_Provision"
}

func (s *ClsProvisioningStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if operation.Cls.Instance.ProvisioningTriggered {
		log.Infof("CLS Provisioning step was already triggered")
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
	operation.Cls.Instance.ProvisioningTriggered = true
	operation, retry := s.operationManager.UpdateOperation(operation)
	if retry > 0 {
		log.Errorf("unable to update operation")
		return operation, time.Second, nil
	}

	return operation, 0, nil
}

func (s *ClsProvisioningStep) provision(smCli servicemanager.Client, operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	var input servicemanager.ProvisioningInput
	input.ID = uuid.New().String()
	input.ServiceID = operation.Cls.Instance.ServiceID
	input.PlanID = operation.Cls.Instance.PlanID
	input.SpaceGUID = uuid.New().String()
	input.OrganizationGUID = uuid.New().String()
	input.Context = map[string]interface{}{
		"platform": "kubernetes",
	}
	input.Parameters = clsParameters{
		RetentionPeriod:    7,
		MaxDataInstances:   2,
		MaxIngestInstances: 2,
		EsAPIEnabled:       false,
	}

	resp, err := smCli.Provision(operation.Cls.Instance.BrokerID, input, true)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("Provision() call failed for brokerID: %s; input: %#v", operation.Cls.Instance.BrokerID, input))
	}
	log.Infof("response from CLS provisioning call: %#v", resp)

	operation.Cls.Instance.InstanceID = input.ID

	return operation, 0, nil
}

func (s *ClsProvisioningStep) handleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg)
}
