package steps

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/new-provisioning-proposal/internal"

	"github.com/sirupsen/logrus"
)

type StepTwo struct {
	count int
}

func (s *StepTwo) Name() string {
	return "Step_Two"
}

func (s *StepTwo) Run(operation internal.ProvisioningOperation, l logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	l.Info(">>>> step process...")
	time.Sleep(1 * time.Second)
	if s.count == 3 {
		operation.EDP = "done"
		l.Info(">>>> step finished")
		return operation, 0, nil
	}

	s.count++
	l.Errorf("step two error")
	return operation, 1 * time.Second, nil
}
