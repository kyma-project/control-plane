package upgrade_kyma

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"
)

var ConfigMapGetter internal.ClusterIDGetter = internal.GetClusterIDWithKubeconfig

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
	clusterID := operation.InstanceDetails.ServiceManagerClusterID
	if clusterID == "" {
		var err error
		if clusterID, err = ConfigMapGetter(operation.InstanceDetails.Kubeconfig); err != nil {
			return s.operationManager.OperationFailed(operation, "failed to create BTP Operator input", err, log)
		}
	}
	creds := operation.ProvisioningParameters.ErsContext.SMOperatorCredentials
	overrides := internal.GetBTPOperatorProvisioningOverrides(creds, clusterID)
	operation.InputCreator.AppendOverrides(internal.BTPOperatorComponentName, overrides)
	operation.InputCreator.EnableOptionalComponent(internal.BTPOperatorComponentName)
	if clusterID == operation.InstanceDetails.ServiceManagerClusterID {
		return operation, 0, nil
	}
	f := func(op *internal.UpgradeKymaOperation) {
		op.InstanceDetails.ServiceManagerClusterID = clusterID
	}
	return s.operationManager.UpdateOperation(operation, f, log)
}
