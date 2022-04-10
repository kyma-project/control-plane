package deprovisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebErrors "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type DeregisterClusterStep struct {
	operationManager   *process.DeprovisionOperationManager
	reconcilerClient   reconciler.Client
	provisionerTimeout time.Duration
}

func NewDeregisterClusterStep(os storage.Operations, cli reconciler.Client) *DeregisterClusterStep {
	return &DeregisterClusterStep{
		operationManager: process.NewDeprovisionOperationManager(os),
		reconcilerClient: cli,
	}
}

func (s *DeregisterClusterStep) Name() string {
	return "Deregister_Cluster"
}

func (s *DeregisterClusterStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if operation.ClusterConfigurationVersion == 0 {
		log.Info("Cluster configuration was not created, skipping")
		return operation, 0, nil
	}
	if operation.ClusterConfigurationDeleted {
		log.Info("Cluster configuration was deleted, skipping")
		return operation, 0, nil
	}
	err := s.reconcilerClient.DeleteCluster(operation.RuntimeID)
	if err != nil {
		return s.handleError(operation, err, log, "cannot remove DataTenant")
	}

	modifiedOp, d, _ := s.operationManager.UpdateOperation(operation, func(op *internal.DeprovisioningOperation) {
		op.ClusterConfigurationDeleted = true
		op.ReconcilerDeregistrationAt = time.Now()
	}, log)

	return modifiedOp, d, nil
}

func (s *DeregisterClusterStep) handleError(operation internal.DeprovisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.DeprovisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)

	if kebErrors.IsTemporaryError(err) {
		since := time.Since(operation.UpdatedAt)
		if since < 30*time.Minute {
			log.Errorf("request to the Reconciler failed: %s. Retry...", err)
			return operation, 15 * time.Second, nil
		}
	}

	log.Errorf("Reconciler cluster configuration have not been deleted in step %s: %s.", s.Name(), err)
	return operation, 0, nil
}
