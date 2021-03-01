package steps

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/new-provisioning-proposal/internal"

	"github.com/sirupsen/logrus"
)

type StepOne struct {
}

func (s *StepOne) Name() string {
	return "Step_One"
}

func (s *StepOne) Run(operation internal.ProvisioningOperation, l logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	l.Info(">>>> step process...")
	time.Sleep(1 * time.Second)
	operation.LMS = "done"

	l.Info(">>>> step finished")
	return operation, 0, nil
}
