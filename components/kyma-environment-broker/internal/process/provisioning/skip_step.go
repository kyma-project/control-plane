package provisioning

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

type SkipStep struct {
	planID           string
	step             Step
	operationManager *process.ProvisionOperationManager
}

func NewSkipStep(os storage.Operations, planID string, step Step) *SkipStep {
	return &SkipStep{
		planID:           planID,
		step:             step,
		operationManager: process.NewProvisionOperationManager(os),
	}
}

func (s *SkipStep) Name() string {
	return s.step.Name()
}

func (s *SkipStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	pp, err := operation.GetProvisioningParameters()
	if err != nil {
		log.Errorf("cannot fetch provisioning parameters from operation: %s", err)
		return s.operationManager.OperationFailed(operation, "invalid operation provisioning parameters")
	}
	if pp.PlanID == s.planID {
		log.Infof("Skipping step %s", s.Name())
		return operation, 0, nil
	}

	return s.step.Run(operation, log)
}
