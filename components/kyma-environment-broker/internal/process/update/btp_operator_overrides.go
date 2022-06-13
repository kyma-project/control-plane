package update

import (
	"reflect"
	"time"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

const BTPOperatorComponentName = "btp-operator"

var ConfigMapGetter internal.ClusterIDGetter = internal.GetClusterIDWithKubeconfig

type BTPOperatorOverridesStep struct {
	operationManager *process.UpdateOperationManager
	components       input.ComponentListProvider
}

func NewBTPOperatorOverridesStep(os storage.Operations, components input.ComponentListProvider) *BTPOperatorOverridesStep {
	return &BTPOperatorOverridesStep{
		operationManager: process.NewUpdateOperationManager(os),
		components:       components,
	}
}

func (s *BTPOperatorOverridesStep) Name() string {
	return "BTPOperatorOverrides"
}

func (s *BTPOperatorOverridesStep) Run(operation internal.UpdatingOperation, logger logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	// get btp-operator component input and calculate overrides
	planName := broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]
	ci, err := getComponentInput(s.components, BTPOperatorComponentName, operation.RuntimeVersion, planName)
	if err != nil {
		return s.operationManager.OperationFailed(operation, "failed to get components", err, logger)
	}
	if err := s.setBTPOperatorOverrides(&ci, operation, logger); err != nil {
		return s.operationManager.OperationFailed(operation, "failed to create BTP Operator input", err, logger)
	}

	// find last btp-operator config if any
	last := -1
	for i, c := range operation.LastRuntimeState.ClusterSetup.KymaConfig.Components {
		if c.Component == BTPOperatorComponentName {
			last = i
			break
		}
	}

	// didn't find btp-operator in last runtime state, append components
	if last == -1 {
		operation.LastRuntimeState.ClusterSetup.KymaConfig.Components = append(operation.LastRuntimeState.ClusterSetup.KymaConfig.Components, ci)
		operation.RequiresReconcilerUpdate = true
		return operation, 0, nil
	}

	// found btp-operator in last runtime state but config isn't matching
	l := operation.LastRuntimeState.ClusterSetup.KymaConfig.Components[last]
	if !reflect.DeepEqual(l, ci) {
		operation.RequiresReconcilerUpdate = true
		operation.LastRuntimeState.ClusterSetup.KymaConfig.Components[last] = ci
	}
	return operation, 0, nil
}

func (s *BTPOperatorOverridesStep) setBTPOperatorOverrides(c *reconcilerApi.Component, operation internal.UpdatingOperation, logger logrus.FieldLogger) error {
	clusterID := operation.InstanceDetails.ServiceManagerClusterID
	if clusterID == "" {
		var err error
		if clusterID, err = ConfigMapGetter(operation.InstanceDetails.Kubeconfig); err != nil {
			return err
		}
	}

	creds := operation.ProvisioningParameters.ErsContext.SMOperatorCredentials
	c.Configuration = internal.GetBTPOperatorReconcilerOverrides(creds, clusterID)
	if clusterID != operation.InstanceDetails.ServiceManagerClusterID {
		f := func(op *internal.UpdatingOperation) {
			op.InstanceDetails.ServiceManagerClusterID = clusterID
		}
		_, _, err := s.operationManager.UpdateOperation(operation, f, logger)
		return err
	}
	return nil
}
