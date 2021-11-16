package upgrade_kyma

import (
	"fmt"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
)

type CreateClusterConfigurationStep struct {
	reconcilerClient    reconciler.Client
	operationManager    *process.UpgradeKymaOperationManager
	provisioningTimeout time.Duration
	runtimeStateStorage storage.RuntimeStates
}

func NewCreateClusterConfiguration(os storage.Operations, runtimeStorage storage.RuntimeStates, reconcilerClient reconciler.Client) *CreateClusterConfigurationStep {
	return &CreateClusterConfigurationStep{
		reconcilerClient:    reconcilerClient,
		operationManager:    process.NewUpgradeKymaOperationManager(os),
		runtimeStateStorage: runtimeStorage,
	}
}

var _ Step = (*CreateClusterConfigurationStep)(nil)

func (s *CreateClusterConfigurationStep) Name() string {
	return "Create_Cluster_Configuration"
}

func (s *CreateClusterConfigurationStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if operation.ClusterConfigurationVersion != 0 {
		log.Debugf("Cluster configuration already created, skipping")
		return operation, 0, nil
	}
	operation.InputCreator.SetRuntimeID(operation.InstanceDetails.RuntimeID).
		SetInstanceID(operation.InstanceID).
		SetShootName(operation.InstanceDetails.ShootName).
		SetShootDomain(operation.ShootDomain).
		SetProvisioningParameters(operation.ProvisioningParameters)

	// enable service management components for upgrade 1.x -> 2.0
	operation.InputCreator.EnableOptionalComponent(provisioning.HelmBrokerComponentName)
	operation.InputCreator.EnableOptionalComponent(provisioning.ServiceCatalogComponentName)
	operation.InputCreator.EnableOptionalComponent(provisioning.ServiceCatalogAddonsComponentName)
	operation.InputCreator.EnableOptionalComponent(provisioning.ServiceManagerComponentName)

	runtimeState, _ := s.runtimeStateStorage.GetLatestByRuntimeID(operation.InstanceDetails.RuntimeID)

	if runtimeState.ClusterSetup == nil {
		for _, component := range runtimeState.KymaConfig.Components {
			// rewrite Component-specific configuration from latest runtimeState
			operation.InputCreator.AppendOverrides(component.Component, component.Configuration)
		}
	}

	if runtimeState.ClusterSetup != nil {
		for _, component := range runtimeState.ClusterSetup.KymaConfig.Components {
			var configList []*gqlschema.ConfigEntryInput
			// rewrite Component-specific configuration from latest runtimeState
			for _, config := range component.Configuration {
				configList = append(configList, &gqlschema.ConfigEntryInput{
					Key:    config.Key,
					Value:  fmt.Sprintf("%v", config.Value),
					Secret: ptr.Bool(config.Secret),
				})

				operation.InputCreator.AppendOverrides(component.Component, configList)
			}
		}
	}

	return operation, 0, nil
}

func (s *CreateClusterConfigurationStep) componentList(cluster reconciler.Cluster) string {
	vals := []string{}
	for _, c := range cluster.KymaConfig.Components {
		vals = append(vals, c.Component)
	}
	return strings.Join(vals, ", ")
}
