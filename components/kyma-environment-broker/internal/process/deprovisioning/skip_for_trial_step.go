package deprovisioning

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

type SkipForTrialPlanStep struct {
	step             Step
	OperationManager *process.DeprovisionOperationManager
}

var _ Step = &SkipForTrialPlanStep{}

func NewSkipForTrialPlanStep(os storage.Operations, step Step) SkipForTrialPlanStep {
	return SkipForTrialPlanStep{
		step:             step,
		OperationManager: process.NewDeprovisionOperationManager(os),
	}
}

func (s SkipForTrialPlanStep) Name() string {
	return s.step.Name()
}

func (s SkipForTrialPlanStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	pp, err := operation.GetProvisioningParameters()
	if err != nil {
		log.Errorf("cannot fetch provisioning parameters from operation: %s", err)
		return s.OperationManager.OperationFailed(operation, "invalid operation provisioning parameters")
	}

	if broker.IsTrialPlan(pp.PlanID) {
		log.Infof("Skipping step %s", s.Name())
		return operation, 0, nil
	}

	return s.step.Run(operation, log)
}
