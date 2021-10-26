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
	if operation.InstanceDetails.SCMigrationTriggered {
		internal.CreateBTPOperatorUpdateInput(operation.InputCreator, operation.ProvisioningParameters.ErsContext.SMOperatorCredentials)
	} else {
		internal.CreateBTPOperatorProvisionInput(operation.InputCreator, operation.ProvisioningParameters.ErsContext.SMOperatorCredentials)
	}
	operation.InputCreator.EnableOptionalComponent(internal.BTPOperatorComponentName)
	operation.InputCreator.DisableOptionalComponent(internal.ServiceManagerComponentName)
	operation.InputCreator.DisableOptionalComponent(internal.HelmBrokerComponentName)
	operation.InputCreator.DisableOptionalComponent(internal.ServiceCatalogComponentName)
	operation.InputCreator.DisableOptionalComponent(internal.ServiceCatalogAddonsComponentName)
	return operation, 0, nil
}
