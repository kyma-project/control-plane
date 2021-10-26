package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/sirupsen/logrus"
)

type BTPOperatorOverridesStep struct{}

func NewBTPOperatorOverridesStep() *BTPOperatorOverridesStep {
	return &BTPOperatorOverridesStep{}
}

func (s *BTPOperatorOverridesStep) Name() string {
	return "BTPOperatorOverrides"
}

func (s *BTPOperatorOverridesStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	operation.InputCreator.CreateBTPOperatorProvisionInput(operation.ProvisioningParameters.ErsContext.SMOperatorCredentials)
	operation.InputCreator.EnableOptionalComponent(input.BTPOperatorComponentName)
	return operation, 0, nil
}
