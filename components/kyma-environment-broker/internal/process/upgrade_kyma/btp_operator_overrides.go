package upgrade_kyma

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"
)

var ConfigMapGetter func(string) internal.ClusterIDGetter = internal.GetClusterIDWithKubeconfig

type BTPOperatorOverridesStep struct {
	operationManager *process.UpgradeKymaOperationManager
}

func NewBTPOperatorOverridesStep(os storage.Operations) *BTPOperatorOverridesStep {
	return &BTPOperatorOverridesStep{
		operationManager: process.NewUpgradeKymaOperationManager(os),
	}
}

func (s *BTPOperatorOverridesStep) Name() string {
	return "BTPOperatorOverrides"
}

func (s *BTPOperatorOverridesStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if err := internal.CreateBTPOperatorProvisionInput(operation.InputCreator, operation.ProvisioningParameters.ErsContext.SMOperatorCredentials, ConfigMapGetter(operation.InstanceDetails.Kubeconfig)); err != nil {
		return s.operationManager.OperationFailed(operation, "failed to create BTP Operator input", err, log)
	}
	operation.InputCreator.EnableOptionalComponent(internal.BTPOperatorComponentName)
	operation.InputCreator.DisableOptionalComponent(internal.ServiceManagerComponentName)
	operation.InputCreator.DisableOptionalComponent(internal.HelmBrokerComponentName)
	operation.InputCreator.DisableOptionalComponent(internal.ServiceCatalogComponentName)
	operation.InputCreator.DisableOptionalComponent(internal.ServiceCatalogAddonsComponentName)
	operation.InputCreator.DisableOptionalComponent(internal.SCMigrationComponentName)
	return operation, 0, nil
}
