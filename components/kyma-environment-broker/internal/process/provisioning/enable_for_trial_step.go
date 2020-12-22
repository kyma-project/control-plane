package provisioning

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
)

type EnableForTrialPlanStep struct {
	step Step
}

// ensure the interface is implemented
var _ Step = (*EnableForTrialPlanStep)(nil)

func NewEnableForTrialPlanStep(step Step) *EnableForTrialPlanStep {
	return &EnableForTrialPlanStep{
		step: step,
	}
}

func (s *EnableForTrialPlanStep) Name() string {
	return s.step.Name()
}

func (s *EnableForTrialPlanStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if broker.IsTrialPlan(operation.ProvisioningParameters.PlanID) {
		log.Infof("Running step %s", s.Name())
		return s.step.Run(operation, log)
	}

	return operation, 0, nil
}
