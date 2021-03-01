package steps

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/new-provisioning-proposal/internal"

	"github.com/sirupsen/logrus"
)

type StepFour struct {
	count int
}

func (s *StepFour) Name() string {
	return "Step_Four"
}

func (s *StepFour) Run(operation internal.ProvisioningOperation, l logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	l.Info(">>>> step process...")
	time.Sleep(3 * time.Second)

	if s.count == 2 {
		operation.Runtime = "done"
		l.Info(">>>> step finished")
		return operation, 0, nil
	}

	s.count++
	l.Errorf("step four error")
	return operation, 3 * time.Second, nil
}
