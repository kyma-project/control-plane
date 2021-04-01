package provisioning

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/sirupsen/logrus"
)

type overrideInputFunc func(*servicemanager.ProvisioningInput) *servicemanager.ProvisioningInput
type infoExtractor func(*internal.ProvisioningOperation) *internal.ServiceManagerInstanceInfo

type SimpleProvisioner struct {
	operationManager  *process.ProvisionOperationManager
	serviceName       string
	infoExtractorFunc infoExtractor
	overrideInputFunc overrideInputFunc
}

type Context interface {
	getProvisioningInput(operation internal.ProvisioningOperation) *servicemanager.ProvisioningInput
}

func NewSimpleProvisioning(serviceName string, info infoExtractor, manager *process.ProvisionOperationManager,
	overrideInput overrideInputFunc) *SimpleProvisioner {

	return &SimpleProvisioner{
		operationManager:  manager,
		serviceName:       serviceName,
		infoExtractorFunc: info,
		overrideInputFunc: overrideInput,
	}
}

func (s *SimpleProvisioner) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	serviceInfo := s.infoExtractorFunc(&operation)
	if serviceInfo.ProvisioningTriggered {
		log.Infof("%s Provisioning step was already triggered", s.serviceName)
		return operation, 0, nil
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.operationManager.HandleError(operation, err, log, fmt.Sprintf("Unable to create Service Manage client"))
	}

	if serviceInfo.InstanceID == "" {
		op, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
			s.infoExtractorFunc(operation).InstanceID = uuid.New().String()
		}, log)
		if retry > 0 {
			log.Errorf("Unable to update operation")
			return operation, time.Second, nil
		}
		operation = op
	}

	// provision
	operation, _, err = s.provision(smCli, operation, log)
	if err != nil {
		return s.operationManager.HandleError(operation, err, log, fmt.Sprintf("provision()  call failed"))
	}

	// save the status
	operation, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
		s.infoExtractorFunc(operation).ProvisioningTriggered = true
	}, log)
	if retry > 0 {
		log.Errorf("unable to update operation")
		return operation, time.Second, nil
	}

	return operation, 0, nil
}

func PassThrough(details *servicemanager.ProvisioningInput) *servicemanager.ProvisioningInput {
	return details
}

func (s *SimpleProvisioner) provision(smCli servicemanager.Client, operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	serviceInfo := s.infoExtractorFunc(&operation)
	input := s.overrideInputFunc(GetSimpleInput(serviceInfo))
	resp, err := smCli.Provision(serviceInfo.BrokerID, *input, true)
	if err != nil {
		return s.operationManager.HandleError(operation, err, log, fmt.Sprintf("Provision() call failed for brokerID: %s; input: %#v", serviceInfo.BrokerID, input))
	}
	log.Debugf("response from %s provisioning call: %#v", s.serviceName, resp)

	return operation, 0, nil
}

func GetSimpleInput(info *internal.ServiceManagerInstanceInfo) *servicemanager.ProvisioningInput {
	var input servicemanager.ProvisioningInput

	input.ID = info.InstanceID
	input.ServiceID = info.ServiceID
	input.PlanID = info.PlanID
	input.SpaceGUID = uuid.New().String()
	input.OrganizationGUID = uuid.New().String()

	input.Context = map[string]interface{}{
		"platform": "kubernetes",
	}
	input.Parameters = map[string]interface{}{}

	return &input
}
