package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type SkipForAzurePlanStep struct {
	step Step
}

func NewSkipForAzurePlanStep(step Step) *SkipForAzurePlanStep {
	return &SkipForAzurePlanStep{
		step: step,
	}
}

func (s *SkipForAzurePlanStep) Name() string {
	return s.step.Name()
}

func (s *SkipForAzurePlanStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	log.Infof("SkipForAzurePlanStep: %#v", operation)
	if broker.IsAzurePlan(operation.ProvisioningParameters.PlanID) {
		log.Infof("Skipping step %s", s.Name())
		//return operation, 0, nil
	} else {
		log.Infof("Don't skip step %s", s.Name())
	}
	return operation, 0, nil
	//return s.step.Run(operation, log)
}
