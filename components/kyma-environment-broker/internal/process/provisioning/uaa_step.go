package provisioning

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"

	"time"
)

const (
	xsuaaOfferingName = "xsuaa"
	xsuaaPlanName     = "application"
)

type UaaInstantiationStep struct {
	operationManager *process.ProvisionOperationManager
}

func NewUaaInstantiationStep(os storage.Operations) *UaaInstantiationStep {
	return &UaaInstantiationStep{
		operationManager: process.NewProvisionOperationManager(os),
	}
}

func (s *UaaInstantiationStep) Name() string {
	return "UAA_POC"
}

func (s *UaaInstantiationStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	// This implementation only lists offerings and plans and it is a placeholder for full implementation

	var cli servicemanager.Client
	cli, err := operation.ServiceManagerClient(log)
	if err != nil {
		log.Infof("SM client creating error: %s", err.Error())
	}

	offs, err := cli.ListOfferings()
	if err != nil {
		log.Infof("List Offerings error: %s", err.Error())
	}
	for _, off := range offs.ServiceOfferings {
		log.Infof("%s %s %s", off.Name, off.BrokerName, off.Description)
	}
	offs, err = cli.ListOfferingsByName(xsuaaOfferingName)
	if err != nil {
		log.Errorf("ListOfferingsByName error: %s", err.Error())
	}
	if offs.IsEmpty() {
		log.Infof("no offerings")
		return operation, 0, nil
	}

	plans, err := cli.ListPlansByName(xsuaaPlanName, offs.ServiceOfferings[0].ID)
	for _, plan := range plans.ServicePlans {
		log.Infof("%s %s %s", plan.Name, plan.Description, plan.Message())
	}

	/*
		1. Provisioning:
		  - find offering and plan
		  - call Provision -> operationID
		    save instanceId and broker ID in the operation




		2. Binding:
		  - checks if provisioning is finished
		  - call Bind
		    save bindingID in the operation
	*/

	return operation, 0, nil
}
