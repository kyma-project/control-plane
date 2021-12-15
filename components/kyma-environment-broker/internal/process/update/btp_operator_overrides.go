package update

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/sirupsen/logrus"
)

const BTPOperatorComponentName = "btp-operator"

var ConfigMapGetter func(string) internal.ClusterIDGetter = internal.GetClusterIDWithKubeconfig

type BTPOperatorOverridesStep struct {
	components input.ComponentListProvider
}

func NewBTPOperatorOverridesStep(components input.ComponentListProvider) *BTPOperatorOverridesStep {
	return &BTPOperatorOverridesStep{
		components: components,
	}
}

func (s *BTPOperatorOverridesStep) Name() string {
	return "BTPOperatorOverrides"
}

func (s *BTPOperatorOverridesStep) Run(operation internal.UpdatingOperation, logger logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	for _, c := range operation.LastRuntimeState.ClusterSetup.KymaConfig.Components {
		if c.Component == BTPOperatorComponentName {
			// already exists
			return operation, 0, nil
		}
	}
	c, err := getComponentInput(s.components, BTPOperatorComponentName, operation.RuntimeVersion)
	if err != nil {
		return operation, 0, err
	}
	if err := s.setBTPOperatorOverrides(&c, operation); err != nil {
		logger.Errorf("failed to get cluster_id from in cluster ConfigMap kyma-system/cluster-info: %v. Retrying in 30s.", err)
		return operation, 30 * time.Second, nil
	}
	operation.LastRuntimeState.ClusterSetup.KymaConfig.Components = append(operation.LastRuntimeState.ClusterSetup.KymaConfig.Components, c)
	operation.RequiresReconcilerUpdate = true
	return operation, 0, nil
}

func (s *BTPOperatorOverridesStep) setBTPOperatorOverrides(c *reconciler.Component, operation internal.UpdatingOperation) error {
	creds := operation.ProvisioningParameters.ErsContext.SMOperatorCredentials
	config, err := internal.GetBTPOperatorReconcilerOverrides(creds, ConfigMapGetter(operation.InstanceDetails.Kubeconfig))
	if err != nil {
		return err
	}
	c.Configuration = config
	return nil
}
