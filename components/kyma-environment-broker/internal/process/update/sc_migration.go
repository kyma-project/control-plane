package update

import (
	"time"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

const (
	SCMigrationComponentName = "sc-migration"
)

type SCMigrationStep struct {
	operationManager *process.UpdateOperationManager
	components       input.ComponentListProvider
}

type SCMigrationFinalizationStep struct {
	reconcilerClient reconciler.Client
}

func NewSCMigrationStep(os storage.Operations, components input.ComponentListProvider) *SCMigrationStep {
	return &SCMigrationStep{
		operationManager: process.NewUpdateOperationManager(os),
		components:       components,
	}
}

func NewSCMigrationFinalizationStep(reconcilerClient reconciler.Client) *SCMigrationFinalizationStep {
	return &SCMigrationFinalizationStep{
		reconcilerClient: reconcilerClient,
	}
}

func (s *SCMigrationStep) Name() string {
	return "SCMigration"
}

func (s *SCMigrationStep) Run(operation internal.UpdatingOperation, logger logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	containsSCMigrationComponent := false
	var components []reconcilerApi.Component
	for _, c := range operation.LastRuntimeState.ClusterSetup.KymaConfig.Components {
		if c.Component != internal.ServiceCatalogComponentName &&
			c.Component != internal.ServiceCatalogAddonsComponentName &&
			c.Component != internal.HelmBrokerComponentName &&
			c.Component != internal.ServiceManagerComponentName {
			components = append(components, c)
		} else {
			// disable reconciler on SVCAT related components so sc-migration can migrate them
			operation.RequiresReconcilerUpdate = true
		}
		if c.Component == SCMigrationComponentName {
			containsSCMigrationComponent = true
		}
	}

	planName := broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]
	if !containsSCMigrationComponent {
		c, err := getComponentInput(s.components, SCMigrationComponentName, operation.RuntimeVersion, planName)
		if err != nil {
			return s.operationManager.OperationFailed(operation, "failed to get components", err, logger)
		}
		components = append(components, c)
		operation.RequiresReconcilerUpdate = true
	}
	operation.LastRuntimeState.ClusterSetup.KymaConfig.Components = components
	return operation, 0, nil
}

func (s *SCMigrationFinalizationStep) Name() string {
	return "SCMigrationFinalization"
}

func (s *SCMigrationFinalizationStep) Run(operation internal.UpdatingOperation, logger logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	components := make([]reconcilerApi.Component, 0, len(operation.LastRuntimeState.ClusterSetup.KymaConfig.Components))
	for _, c := range operation.LastRuntimeState.ClusterSetup.KymaConfig.Components {
		if c.Component != internal.ServiceCatalogComponentName &&
			c.Component != internal.ServiceCatalogAddonsComponentName &&
			c.Component != internal.HelmBrokerComponentName &&
			c.Component != internal.SCMigrationComponentName &&
			c.Component != internal.ServiceManagerComponentName {
			components = append(components, c)
		} else {
			operation.RequiresReconcilerUpdate = true
		}
	}
	operation.LastRuntimeState.ClusterSetup.KymaConfig.Components = components
	return operation, 0, nil
}
