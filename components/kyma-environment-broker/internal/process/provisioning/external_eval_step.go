package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type ExternalEvalStep struct {
	externalEvalCreator *ExternalEvalCreator
}

// ensure the interface is implemented
var _ Step = (*ExternalEvalStep)(nil)

func NewExternalEvalStep(externalEvalCreator *ExternalEvalCreator) *ExternalEvalStep {
	return &ExternalEvalStep{
		externalEvalCreator: externalEvalCreator,
	}
}

func (e ExternalEvalStep) Name() string {
	return "AVS_Create_External_Eval_Step"
}

func (s *ExternalEvalStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if broker.IsTrialPlan(operation.ProvisioningParameters.PlanID) || broker.IsFreemiumPlan(operation.ProvisioningParameters.PlanID) {
		log.Debug("skipping AVS external evaluation creation for trial/freemium plan")
		return operation, 0, nil
	}

	op, repeat, err := s.externalEvalCreator.createEval(operation, operation.DashboardURL, log)
	if err != nil || repeat != 0 {
		return operation, repeat, err
	}
	return op, 0, nil
}
