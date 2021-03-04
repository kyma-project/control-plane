package deprovisioning

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type SkipForAllStep struct {
	step Step
}

var _ Step = &SkipForAllStep{}

func NewSkipForAllStep(step Step) SkipForAllStep {
	return SkipForAllStep{
		step: step,
	}
}

func (s SkipForAllStep) Name() string {
	return s.step.Name()
}

func (s SkipForAllStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	log.Infof("SkipForAllStep: %#v", operation)
	log.Infof("Skipping step %s", s.Name())

	return operation, 0, nil
}
