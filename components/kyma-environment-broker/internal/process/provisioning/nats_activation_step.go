package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	pkgErrors "github.com/pkg/errors"

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
	// run the step only if Kyma<1.21  && (IsAzure==false || IsTrial==true)
	kymaVersion := operation.ProvisioningParameters.Parameters.KymaVersion
	atLeast_1_21, err := cls.IsKymaVersionAtLeast_1_21(kymaVersion)
	if err != nil {
		log.Error(pkgErrors.Wrapf(err, "while checking Kyma version"))
		return operation, 0, nil
	}
	if atLeast_1_21 {
		log.Infof("Skipping step %s for Kyma version %s", s.Name(), kymaVersion)
		return operation, 0, nil
	}
	if planID := operation.ProvisioningParameters.PlanID; broker.IsAzurePlan(planID) && !broker.IsTrialPlan(planID) {
		log.Infof("Skipping step %s for planID %s", s.Name(), planID)
		return operation, 0, nil
	}
	// run the step
	return s.step.Run(operation, log)
}
