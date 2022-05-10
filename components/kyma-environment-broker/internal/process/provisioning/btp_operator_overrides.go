package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

var ConfigMapGetter func(string) internal.ClusterIDGetter = internal.GetClusterIDWithKubeconfig

type BTPOperatorOverridesStep struct {
	operationManager *process.ProvisionOperationManager
}

func NewBTPOperatorOverridesStep(os storage.Operations) *BTPOperatorOverridesStep {
	return &BTPOperatorOverridesStep{
		operationManager: process.NewProvisionOperationManager(os),
	}
}

func (s *BTPOperatorOverridesStep) Name() string {
	return "BTPOperatorOverrides"
}

func (s *BTPOperatorOverridesStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if err := internal.CreateBTPOperatorProvisionInput(operation.InputCreator, operation.ProvisioningParameters.ErsContext.SMOperatorCredentials, ConfigMapGetter(operation.InstanceDetails.Kubeconfig)); err != nil {
		return s.operationManager.OperationFailed(operation, "failed to create BTP Operator input", err, log)
	}
	operation.InputCreator.EnableOptionalComponent(internal.BTPOperatorComponentName)
	return operation, 0, nil
}
