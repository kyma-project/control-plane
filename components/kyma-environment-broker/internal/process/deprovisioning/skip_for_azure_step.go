package deprovisioning

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
)

type SkipForAzurePlanStep struct {
	step Step
}

var _ Step = &SkipForAzurePlanStep{}

func NewSkipForAzurePlanStep(step Step) SkipForAzurePlanStep {
	return SkipForAzurePlanStep{
		step: step,
	}
}

func (s SkipForAzurePlanStep) Name() string {
	return s.step.Name()
}

func (s SkipForAzurePlanStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if broker.IsAzurePlan(operation.ProvisioningParameters.PlanID) {
		log.Infof("Skipping step %s", s.Name())
		return operation, 0, nil
	}

	return s.step.Run(operation, log)
}
