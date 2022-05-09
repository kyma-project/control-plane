package update

import (
	"reflect"
	"time"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

const BTPOperatorComponentName = "btp-operator"

var ConfigMapGetter func(string) internal.ClusterIDGetter = internal.GetClusterIDWithKubeconfig

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
	ci, err := getComponentInput(s.components, BTPOperatorComponentName, operation.RuntimeVersion)
	if err != nil {
		return s.operationManager.OperationFailed(operation, "failed to get components", err, logger)
	}
	if err := s.setBTPOperatorOverrides(&ci, operation); err != nil {
		logger.Errorf("failed to get cluster_id from in cluster ConfigMap kyma-system/cluster-info: %v. Retrying in 30s.", err)
		return operation, 30 * time.Second, nil
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

func (s *BTPOperatorOverridesStep) setBTPOperatorOverrides(c *reconcilerApi.Component, operation internal.UpdatingOperation) error {
	creds := operation.ProvisioningParameters.ErsContext.SMOperatorCredentials
	config, err := internal.GetBTPOperatorReconcilerOverrides(creds, ConfigMapGetter(operation.InstanceDetails.Kubeconfig))
	if err != nil {
		return err
	}
	c.Configuration = config
	return nil
}
