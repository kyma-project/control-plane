package provisioning

import (
	"time"

	pkgErrors "github.com/pkg/errors"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/version"
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
	// run the step only if  KymaVersion<1.21 && IsAzure==true && IsTrial==false
	kymaVersion := operation.RuntimeVersion.Version
	atLeast_1_21, err := version.IsKymaVersionAtLeast_1_21(kymaVersion)
	if err != nil {
		log.Error(pkgErrors.Wrapf(err, "while checking Kyma version"))
		return operation, 0, nil
	}
	if atLeast_1_21 {
		log.Infof("Skipping step %s for Kyma version %s", s.Name(), kymaVersion)
		return operation, 0, nil
	}
	if planID := operation.ProvisioningParameters.PlanID; !broker.IsAzurePlan(planID) || broker.IsTrialPlan(planID) {
		log.Infof("Skipping step %s for planID %s", s.Name(), planID)
		return operation, 0, nil
	}
	// run the step
	return s.step.Run(operation, log)
}
