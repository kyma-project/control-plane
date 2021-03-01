package steps

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/new-provisioning-proposal/internal"

	"github.com/sirupsen/logrus"
)

type StepFive struct{}

func (s *StepFive) Name() string {
	return "Step_Five"
}

func (s *StepFive) Run(operation internal.ProvisioningOperation, l logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	l.Info(">>>> step process...")
	time.Sleep(1 * time.Second)

	if operation.Runtime == "done" {
		l.Info(">>>> step finished")
		return operation, 0, nil
	}

	l.Errorf("operation is not ready")
	return operation, 1 * time.Second, nil
}
