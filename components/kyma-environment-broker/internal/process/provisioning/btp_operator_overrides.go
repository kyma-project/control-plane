package provisioning

import (
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type BTPOperatorOverridesStep struct {
	operationManager *process.OperationManager
}

func NewBTPOperatorOverridesStep(os storage.Operations) *BTPOperatorOverridesStep {
	return &BTPOperatorOverridesStep{
		operationManager: process.NewOperationManager(os),
	}
}

func (s *BTPOperatorOverridesStep) Name() string {
	return "BTPOperatorOverrides"
}

func (s *BTPOperatorOverridesStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	clusterID := uuid.NewString()
	overrides := internal.GetBTPOperatorProvisioningOverrides(operation.ProvisioningParameters.ErsContext.SMOperatorCredentials, clusterID)
	f := func(op *internal.Operation) {
		op.InstanceDetails.ServiceManagerClusterID = clusterID
	}
	operation.InputCreator.AppendOverrides(internal.BTPOperatorComponentName, overrides)
	operation.InputCreator.EnableOptionalComponent(internal.BTPOperatorComponentName)
	return s.operationManager.UpdateOperation(operation, f, log)
}
