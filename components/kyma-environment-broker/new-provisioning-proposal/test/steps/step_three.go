package steps

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/new-provisioning-proposal/internal"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type StepThree struct {
}

func (s *StepThree) Name() string {
	return "Step_Three"
}

func (s *StepThree) Run(operation internal.ProvisioningOperation, l logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	l.Info(">>>> step process...")
	time.Sleep(1 * time.Second)

	if operation.LMS == "done" && operation.EDP == "done" {
		l.Info(">>>> step finished")
		return operation, 0, nil
	}

	l.Errorf("steps before didn't do their job")
	return operation, 0, errors.New("step three failed")
}
