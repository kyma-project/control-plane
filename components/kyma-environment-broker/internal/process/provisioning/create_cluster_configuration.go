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
}

func NewCreateClusterConfiguration(os storage.Operations,
	reconcilerClient reconciler.Client) *CreateClusterConfigurationStep {
	return &CreateClusterConfigurationStep{
		reconcilerClient: reconcilerClient,
		operationManager: process.NewProvisionOperationManager(os),
	}
}

var _ Step = (*CreateClusterConfigurationStep)(nil)

func (s *CreateClusterConfigurationStep) Name() string {
	return "create_cluster_configuration"
}

func (s *CreateClusterConfigurationStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	operation.InputCreator.SetRuntimeID(operation.RuntimeID).
		SetInstanceID(operation.InstanceID).
		SetKubeconfig(operation.Kubeconfig)

	clusterConfigurtation, err := operation.InputCreator.CreateClusterConfiguration()
	if err != nil {
		log.Errorf("Unable to create cluster configuration: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "invalid operation data - cannot create cluster configuration", log)
	}

	log.Infof("Creating Cluster Configuration: cluster(runtimeID)=%s, kymaVersion=%s, kymaProfile=%s, components=[%s]",
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
