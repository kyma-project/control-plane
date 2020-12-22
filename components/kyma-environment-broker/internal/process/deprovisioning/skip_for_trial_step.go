package deprovisioning

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
)

type SkipForTrialPlanStep struct {
	step Step
}

var _ Step = &SkipForTrialPlanStep{}

func NewSkipForTrialPlanStep(step Step) SkipForTrialPlanStep {
	return SkipForTrialPlanStep{
		step: step,
	}
}

func (s SkipForTrialPlanStep) Name() string {
	return s.step.Name()
}

func (s SkipForTrialPlanStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if broker.IsTrialPlan(operation.ProvisioningParameters.PlanID) {
		log.Infof("Skipping step %s", s.Name())
		return operation, 0, nil
	}

	return s.step.Run(operation, log)
}
