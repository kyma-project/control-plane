package provisioning

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
)

type AzureEventHubActivationStep struct {
	step Step
}

func NewAzureEventHubActivationStep(step Step) *AzureEventHubActivationStep {
	return &AzureEventHubActivationStep{
		step: step,
	}
}

func (s *AzureEventHubActivationStep) Name() string {
	return s.step.Name()
}

func (s *AzureEventHubActivationStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if broker.IsTrialPlan(operation.ProvisioningParameters.PlanID) ||
		operation.ProvisioningParameters.PlanID == broker.AWSPlanID ||
		operation.ProvisioningParameters.PlanID == broker.OpenStackPlanID {
		log.Infof("Skipping step %s", s.Name())
		return operation, 0, nil
	}

	return s.step.Run(operation, log)
}
