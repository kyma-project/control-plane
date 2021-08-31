package upgrade_kyma

import (
	"fmt"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type ApplyClusterConfigurationStep struct {
	operationManager *process.UpgradeKymaOperationManager
	reconcilerClient reconciler.Client
}

func NewApplyClusterConfigurationStep(os storage.Operations,
	reconcilerClient reconciler.Client) *ApplyClusterConfigurationStep {
	return &ApplyClusterConfigurationStep{
		reconcilerClient: reconcilerClient,
		operationManager: process.NewUpgradeKymaOperationManager(os),
	}
}

func (s *ApplyClusterConfigurationStep) Name() string {
	return "Overrides_From_Secrets_And_Config_Step"
}

func (s *ApplyClusterConfigurationStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	operation.InputCreator.SetRuntimeID(operation.Runtime.RuntimeID).
		SetInstanceID(operation.InstanceID).
		SetKubeconfig(operation.Kubeconfig)

	clusterConfigurtation, err := operation.InputCreator.CreateClusterConfiguration()
	if err != nil {
		log.Errorf("Unable to apply cluster configuration: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "invalid operation data - cannot create cluster configuration", log)
	}

	log.Infof("Apply Cluster Configuration: cluster(runtimeID)=%s, kymaVersion=%s, kymaProfile=%s, components=[%s]",
		clusterConfigurtation.Cluster,
		clusterConfigurtation.KymaConfig.Version,
		clusterConfigurtation.KymaConfig.Profile,
		s.componentList(clusterConfigurtation))
	state, err := s.reconcilerClient.ApplyClusterConfig(clusterConfigurtation)
	switch {
	case kebError.IsTemporaryError(err):
		msg := fmt.Sprintf("Request to Reconciler failed: %s", err.Error())
		log.Error(msg)
		return operation, 5 * time.Second, nil
	case err != nil:
		msg := fmt.Sprintf("Request to Reconciler failed: %s", err.Error())
		log.Error(msg)
		return s.operationManager.OperationFailed(operation, msg, log)
	}
	log.Infof("Cluster configuration version %d", state.ConfigurationVersion)

	updatedOperation, repeat := s.operationManager.UpdateOperation(operation, func(operation *internal.UpgradeKymaOperation) {
		operation.ClusterConfigurationVersion = state.ConfigurationVersion
	}, log)
	if repeat != 0 {
		log.Errorf("cannot save cluster configuration version")
		return operation, 5 * time.Second, nil
	}

	return updatedOperation, 0, nil

}

func (s *ApplyClusterConfigurationStep) componentList(cluster reconciler.Cluster) string {
	vals := []string{}
	for _, c := range cluster.KymaConfig.Components {
		vals = append(vals, c.Component)
	}
	return strings.Join(vals, ", ")
}
