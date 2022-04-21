package upgrade_kyma

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

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
	internal.CreateBTPOperatorProvisionInput(operation.InputCreator, operation.ProvisioningParameters.ErsContext.SMOperatorCredentials)
	operation.InputCreator.EnableOptionalComponent(internal.BTPOperatorComponentName)
	operation.InputCreator.DisableOptionalComponent(internal.ServiceManagerComponentName)
	operation.InputCreator.DisableOptionalComponent(internal.HelmBrokerComponentName)
	operation.InputCreator.DisableOptionalComponent(internal.ServiceCatalogComponentName)
	operation.InputCreator.DisableOptionalComponent(internal.ServiceCatalogAddonsComponentName)
	operation.InputCreator.DisableOptionalComponent(internal.SCMigrationComponentName)
	return operation, 0, nil
}
