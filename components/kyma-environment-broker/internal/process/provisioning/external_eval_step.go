package provisioning

import (
	"fmt"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type ExternalEvalStep struct {
	externalEvalCreator *ExternalEvalCreator
}

// ensure the interface is implemented
var _ process.Step = (*ExternalEvalStep)(nil)

func NewExternalEvalStep(externalEvalCreator *ExternalEvalCreator) *ExternalEvalStep {
	return &ExternalEvalStep{
		externalEvalCreator: externalEvalCreator,
	}
}

func (e ExternalEvalStep) Name() string {
	return "AVS_Create_External_Eval_Step"
}

func (s *ExternalEvalStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if broker.IsTrialPlan(operation.ProvisioningParameters.PlanID) || broker.IsFreemiumPlan(operation.ProvisioningParameters.PlanID) {
		log.Debug("skipping AVS external evaluation creation for trial/freemium plan")
		return operation, 0, nil
	}

	targetURL := fmt.Sprintf("https://healthz.%s/healthz/ready ", operation.ShootDomain)
	op, repeat, err := s.externalEvalCreator.createEval(operation, targetURL, log)
	if err != nil || repeat != 0 {
		return operation, repeat, err
	}
	return op, 0, nil
}
