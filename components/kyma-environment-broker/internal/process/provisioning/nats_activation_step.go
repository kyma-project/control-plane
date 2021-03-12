package provisioning

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
)

type NatsActivationStep struct {
	step Step
}

// ensure the interface is implemented
var _ Step = (*NatsActivationStep)(nil)

func NewNatsActivationStep(step Step) *NatsActivationStep {
	return &NatsActivationStep{
		step: step,
	}
}

func (s *NatsActivationStep) Name() string {
	return s.step.Name()
}

func (s *NatsActivationStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if broker.IsTrialPlan(operation.ProvisioningParameters.PlanID) ||
		operation.ProvisioningParameters.PlanID == broker.AWSPlanID ||
		operation.ProvisioningParameters.PlanID == broker.OpenStackPlanID {
		log.Infof("Running step %s", s.Name())
		return s.step.Run(operation, log)
	}

	return operation, 0, nil
}
