package upgrade_kyma

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

func (s *BTPOperatorOverridesStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	operation.InputCreator.CreateBTPOperatorUpdateInput(operation.ProvisioningParameters.ErsContext.SMOperatorCredentials)
	operation.InputCreator.EnableOptionalComponent(input.BTPOperatorComponentName)
	operation.InputCreator.DisableOptionalComponent(input.ServiceManagerComponentName)
	operation.InputCreator.DisableOptionalComponent(input.HelmBrokerComponentName)
	operation.InputCreator.DisableOptionalComponent(input.ServiceCatalogComponentName)
	operation.InputCreator.DisableOptionalComponent(input.ServiceCatalogAddonsComponentName)
	return operation, 0, nil
}
