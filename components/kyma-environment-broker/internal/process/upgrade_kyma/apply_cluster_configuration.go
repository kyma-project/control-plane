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
	operationManager    *process.UpgradeKymaOperationManager
	reconcilerClient    reconciler.Client
	runtimeStateStorage storage.RuntimeStates
}

func NewApplyClusterConfigurationStep(os storage.Operations, rs storage.RuntimeStates, reconcilerClient reconciler.Client) *ApplyClusterConfigurationStep {
	return &ApplyClusterConfigurationStep{
		operationManager:    process.NewUpgradeKymaOperationManager(os),
		reconcilerClient:    reconcilerClient,
		runtimeStateStorage: rs,
	}
}

func (s *ApplyClusterConfigurationStep) Name() string {
	return "Apply_Cluster_Configuration"
}

func (s *ApplyClusterConfigurationStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	operation.InputCreator.DisableOptionalComponent(internal.SCMigrationComponentName)
	operation.InputCreator.SetRuntimeID(operation.InstanceDetails.RuntimeID).
		SetInstanceID(operation.InstanceID).
		SetShootName(operation.InstanceDetails.ShootName).
		SetShootDomain(operation.ShootDomain).
		SetProvisioningParameters(operation.ProvisioningParameters)

	clusterConfiguration, err := operation.InputCreator.CreateClusterConfiguration()
	if err != nil {
		log.Errorf("Unable to apply cluster configuration: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "invalid operation data - cannot create cluster configuration", log)
	}

	err = s.runtimeStateStorage.Insert(
		internal.NewRuntimeStateWithReconcilerInput(clusterConfiguration.Cluster, operation.Operation.ID, &clusterConfiguration))
	if err != nil {
		log.Errorf("cannot insert runtimeState with reconciler payload: %s", err)
		return operation, 10 * time.Second, nil
	}

	log.Infof("Apply Cluster Configuration: cluster(runtimeID)=%s, kymaVersion=%s, kymaProfile=%s, components=[%s]",
		clusterConfiguration.Cluster,
		clusterConfiguration.KymaConfig.Version,
		clusterConfiguration.KymaConfig.Profile,
		s.componentList(clusterConfiguration))
	state, err := s.reconcilerClient.ApplyClusterConfig(clusterConfiguration)
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

	// return some retry value to get back to initialisation step
	return updatedOperation, 5 * time.Second, nil

}

func (s *ApplyClusterConfigurationStep) componentList(cluster reconciler.Cluster) string {
	vals := []string{}
	for _, c := range cluster.KymaConfig.Components {
		vals = append(vals, c.Component)
	}
	return strings.Join(vals, ", ")
}
