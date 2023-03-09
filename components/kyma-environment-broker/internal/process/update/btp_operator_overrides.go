package update

import (
	"time"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

const BTPOperatorComponentName = "btp-operator"

var ConfigMapGetter internal.ClusterIDGetter = internal.GetClusterIDWithKubeconfig

type BTPOperatorOverridesStep struct {
	operationManager *process.OperationManager
	components       input.ComponentListProvider
}

func NewBTPOperatorOverridesStep(os storage.Operations, components input.ComponentListProvider) *BTPOperatorOverridesStep {
	return &BTPOperatorOverridesStep{
		operationManager: process.NewOperationManager(os),
		components:       components,
	}
}

func (s *BTPOperatorOverridesStep) Name() string {
	return "BTPOperatorOverrides"
}

func (s *BTPOperatorOverridesStep) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if operation.LastRuntimeState.ClusterSetup == nil {
		logger.Infof("no last runtime state found, skipping")
		return operation, 0, nil
	}
	// get btp-operator component input and calculate overrides
	ci, err := getComponentInput(s.components, BTPOperatorComponentName, operation.RuntimeVersion, operation.InputCreator.Configuration())
	if err != nil {
		return s.operationManager.RetryOperation(operation, "failed to get components", err, 5*time.Second, 30*time.Second, logger)
	}
	if err := s.setBTPOperatorOverrides(&ci, operation, logger); err != nil {
		return s.operationManager.RetryOperation(operation, "failed to create BTP Operator input", err, 5*time.Second, 30*time.Second, logger)
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
	if !internal.CheckBTPCredsMatching(l, ci) {
		operation.RequiresReconcilerUpdate = true
		operation.LastRuntimeState.ClusterSetup.KymaConfig.Components[last] = ci
	}
	return operation, 0, nil
}

func (s *BTPOperatorOverridesStep) setBTPOperatorOverrides(c *reconcilerApi.Component, operation internal.Operation, logger logrus.FieldLogger) error {
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
		f := func(op *internal.Operation) {
			op.InstanceDetails.ServiceManagerClusterID = clusterID
		}
		_, _, err := s.operationManager.UpdateOperation(operation, f, logger)
		return err
	}
	return nil
}
