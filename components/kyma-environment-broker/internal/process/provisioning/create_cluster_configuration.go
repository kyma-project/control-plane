package provisioning

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

type CreateClusterConfigurationStep struct {
	reconcilerClient    reconciler.Client
	operationManager    *process.ProvisionOperationManager
	provisioningTimeout time.Duration
	runtimeStateStorage storage.RuntimeStates
}

func NewCreateClusterConfiguration(os storage.Operations, runtimeStorage storage.RuntimeStates, reconcilerClient reconciler.Client) *CreateClusterConfigurationStep {
	return &CreateClusterConfigurationStep{
		reconcilerClient:    reconcilerClient,
		operationManager:    process.NewProvisionOperationManager(os),
		runtimeStateStorage: runtimeStorage,
	}
}

var _ Step = (*CreateClusterConfigurationStep)(nil)

func (s *CreateClusterConfigurationStep) Name() string {
	return "Create_Cluster_Configuration"
}

func (s *CreateClusterConfigurationStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if operation.ClusterConfigurationVersion != 0 {
		log.Debugf("Cluster configuration already created, skipping")
		return operation, 0, nil
	}
	operation.InputCreator.SetRuntimeID(operation.RuntimeID).
		SetInstanceID(operation.InstanceID).
		SetKubeconfig(operation.Kubeconfig).
		SetShootName(operation.ShootName).
		SetShootDomain(operation.ShootDomain).
		SetShootDNSProviders(operation.ShootDNSProviders).
		SetProvisioningParameters(operation.ProvisioningParameters)

	clusterConfiguration, err := operation.InputCreator.CreateClusterConfiguration()
	if err != nil {
		log.Errorf("Unable to create cluster configuration: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "invalid operation data - cannot create cluster configuration", log)
	}

	err = s.runtimeStateStorage.Insert(
		internal.NewRuntimeStateWithReconcilerInput(clusterConfiguration.Cluster, operation.ID, &clusterConfiguration))
	if err != nil {
		log.Errorf("cannot insert runtimeState with reconciler payload: %s", err)
		return operation, 10 * time.Second, nil
	}

	log.Infof("Creating Cluster Configuration: cluster(runtimeID)=%s, kymaVersion=%s, kymaProfile=%s, components=[%s]",
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

	updatedOperation, repeat := s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
		operation.ClusterConfigurationVersion = state.ConfigurationVersion
	}, log)
	if repeat != 0 {
		log.Errorf("cannot save cluster configuration version")
		return operation, 5 * time.Second, nil
	}

	return updatedOperation, 0, nil
}

func (s *CreateClusterConfigurationStep) componentList(cluster reconciler.Cluster) string {
	vals := []string{}
	for _, c := range cluster.KymaConfig.Components {
		vals = append(vals, c.Component)
	}
	return strings.Join(vals, ", ")
}
