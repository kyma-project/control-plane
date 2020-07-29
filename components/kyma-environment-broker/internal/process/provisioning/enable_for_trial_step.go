package provisioning

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

type EnableForTrialPlanStep struct {
	step             Step
	operationManager *process.ProvisionOperationManager
}

// ensure the interface is implemented
var _ Step = (*EnableForTrialPlanStep)(nil)

func NewEnableForTrialPlanStep(os storage.Operations, step Step) *EnableForTrialPlanStep {
	return &EnableForTrialPlanStep{
		step:             step,
		operationManager: process.NewProvisionOperationManager(os),
	}
}

func (s *EnableForTrialPlanStep) Name() string {
	return s.step.Name()
}

func (s *EnableForTrialPlanStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	pp, err := operation.GetProvisioningParameters()
	if err != nil {
		log.Errorf("cannot fetch provisioning parameters from operation: %s", err)
		return s.operationManager.OperationFailed(operation, "invalid operation provisioning parameters")
	}
	if broker.IsTrialPlan(pp.PlanID) {
		log.Infof("Enabling step %s", s.Name())
		//return operation, 0, nil
		return s.step.Run(operation, log)
	}

	//return s.step.Run(operation, log)
	return operation, 0, nil
}
