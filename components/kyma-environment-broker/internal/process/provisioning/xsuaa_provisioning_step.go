package provisioning

import (
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	uaa "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager/xsuaa"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type XSUAAProvisioningStep struct {
	operationManager  *process.ProvisionOperationManager
	parametersFactory *uaa.ParametersFactory
}

func NewXSUAAProvisioningStep(repo storage.Operations, cfg uaa.Config) *XSUAAProvisioningStep {
	return &XSUAAProvisioningStep{
		operationManager:  process.NewProvisionOperationManager(repo),
		parametersFactory: uaa.NewParametersFactory(cfg),
	}
}

func (s *XSUAAProvisioningStep) Name() string {
	return "XSUAA_Provisioning"
}

func (s *XSUAAProvisioningStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if operation.XSUAA.Instance.ProvisioningTriggered {
		return operation, 0, nil
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, "unable to create Service Manager client", log)
	}
	instanceInfo := operation.XSUAA.Instance
	if operation.ShootDomain == "" {
		log.Errorf("ShootDomain is not set in the operation")
		// this may happen, when a provisioning is started with a version which does not set the Domain
		// then KEB is restarted to a newer version
		return s.operationManager.OperationFailed(operation, "The `ShootDomain` must be set in the operation, but it is empty")
	}

	// first try to save the instance ID then perform provisioning to be sure we do not loose provisioned Id
	// We can always deprovision not existing instance and get http 410 which is handled correctly by the client
	// more: https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#deprovisioning
	retry := time.Duration(0)
	operation, retry = s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
		if operation.XSUAA.Instance.InstanceID == "" {
			operation.XSUAA.Instance.InstanceID = uuid.New().String()
		}
		if operation.XSUAA.XSAppname == "" {
			operation.XSUAA.XSAppname = uaa.XSAppname(operation.ShootDomain)
		}
	})
	if retry > 0 {
		return operation, time.Second, nil
	}

	log.Infof("Trying to provision: brokerID=%s, serviceID=%s, planID=%s, instanceID=%s",
		instanceInfo.BrokerID, instanceInfo.ServiceID, instanceInfo.PlanID, operation.XSUAA.Instance.InstanceID)

	parameters := s.parametersFactory.Generate(operation.ShootName, operation.ShootDomain, operation.XSUAA.XSAppname)

	s.parametersFactory.Generate(operation.ShootName, operation.ShootDomain, operation.XSUAA.XSAppname)
	resp, err := smCli.Provision(instanceInfo.BrokerID, servicemanager.ProvisioningInput{
		ProvisionRequest: servicemanager.ProvisionRequest{
			ServiceID:  instanceInfo.ServiceID,
			PlanID:     instanceInfo.PlanID,
			Parameters: parameters,
			Context: map[string]interface{}{
				"platform": "kubernetes",
			},
			OrganizationGUID: uuid.New().String(), // todo: correct values?
			SpaceGUID:        uuid.New().String(), // todo: correct values?
		},
		ID: operation.XSUAA.Instance.InstanceID,
	}, true)
	if err != nil {
		return s.handleError(operation, err, "unable to provision XSUAA instance", log)
	}

	operation, retry = s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
		operation.XSUAA.Instance.ProvisioningTriggered = true
		if resp.IsDone() {
			operation.XSUAA.Instance.Provisioned = true
		}
	})
	if retry > 0 {
		return operation, time.Second, nil
	}

	return operation, 0, nil
}

func (s *XSUAAProvisioningStep) handleError(operation internal.ProvisioningOperation, err error, msg string, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	switch {
	case kebError.IsTemporaryError(err):
		return s.operationManager.RetryOperation(operation, msg, 10*time.Second, time.Minute*30, log)
	default:
		return s.operationManager.OperationFailed(operation, msg)
	}
}
