package provisioning

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type ClsActivationStep struct {
	disabled bool
	step     Step
}

func NewClsActivationStep(disabled bool, step Step) *ClsActivationStep {
	return &ClsActivationStep{
		disabled: disabled,
		step:     step,
	}
}

func (s *ClsActivationStep) Name() string {
	return s.step.Name()
}

func (s *ClsActivationStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {

	if s.disabled {
		return operation, 0, nil
	}

	return s.step.Run(operation, log)
}
