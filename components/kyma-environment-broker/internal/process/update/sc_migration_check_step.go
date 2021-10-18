package update

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/sirupsen/logrus"
)

const (
	ServiceCatalogComponentName       = "service-catalog"
	ServiceCatalogAddonsComponentName = "service-catalog-addons"
	HelmBrokerComponentName           = "heml-broker"
)

type SCMigrationCheckStep struct {
	reconcilerClient reconciler.Client
}

func NewCheckSCMigrationDone(reconcilerClient reconciler.Client) *SCMigrationCheckStep {
	return &SCMigrationCheckStep{
		reconcilerClient: reconcilerClient,
	}
}

func (s *SCMigrationCheckStep) Name() string {
	return "SCMigrationCheck"
}

func (s *SCMigrationCheckStep) Run(operation internal.UpdatingOperation, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	state, err := s.reconcilerClient.GetCluster(operation.RuntimeID, operation.ClusterConfigurationVersion)

	if kebError.IsTemporaryError(err) {
		log.Errorf("Reconciler GetCluster method failed (temporary error, retrying): %v", err)
		return operation, 1 * time.Minute, nil
	} else if err != nil {
		log.Errorf("Reconciler GetCluster method failed: %v", err)
		return operation, 0, fmt.Errorf("unable to get cluster state: %v", err)
	}
	switch state.Status {
	case reconciler.ClusterStatusReconciling, reconciler.ClusterStatusPending:
		return operation, 30 * time.Second, nil
	case reconciler.ClusterStatusReady:
		s.removeServiceCatalog(&operation)
		return operation, 0, nil
	case reconciler.ClusterStatusError:
		errMsg := fmt.Sprintf("Reconciler failed. %v", state.PrettyFailures())
		log.Warnf(errMsg)
		return operation, 0, fmt.Errorf(errMsg)
	default:
		log.Warnf("Unknown reconciler cluster state: %v", state.Status)
		return operation, 0, fmt.Errorf("Reconciler error")
	}
}

func (s *SCMigrationCheckStep) removeServiceCatalog(operation *internal.UpdatingOperation) {
	components := make([]reconciler.Components, 0, len(operation.LastRuntimeState.ClusterSetup.KymaConfig.Components))
	for _, c := range operation.LastRuntimeState.ClusterSetup.KymaConfig.Components {
		if c.Component != ServiceCatalogComponentName &&
			c.Component != ServiceCatalogAddonsComponentName &&
			c.Component != HelmBrokerComponentName &&
			c.Component != SCMigrationComponentName {
			components = append(components, c)
		}
	}
	operation.LastRuntimeState.ClusterSetup.KymaConfig.Components = components
}
