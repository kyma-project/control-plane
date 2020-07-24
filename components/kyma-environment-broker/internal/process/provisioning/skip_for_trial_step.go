package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

type SkipForTrialPlanStep struct {
	step             Step
	operationManager *process.ProvisionOperationManager
}

func NewSkipForTrialPlanStep(os storage.Operations, step Step) *SkipForTrialPlanStep {
	return &SkipForTrialPlanStep{
		step:             step,
		operationManager: process.NewProvisionOperationManager(os),
	}
}

func (s *SkipForTrialPlanStep) Name() string {
	return s.step.Name()
}

func (s *SkipForTrialPlanStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	pp, err := operation.GetProvisioningParameters()
	if err != nil {
		log.Errorf("cannot fetch provisioning parameters from operation: %s", err)
		return s.operationManager.OperationFailed(operation, "invalid operation provisioning parameters")
	}
	if broker.IsTrialPlan(pp.PlanID) {
		log.Infof("Skipping step %s", s.Name())
		return operation, 0, nil
	}

	return s.step.Run(operation, log)
}
