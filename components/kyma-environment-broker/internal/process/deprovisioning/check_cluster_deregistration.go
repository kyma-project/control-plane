package deprovisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/sirupsen/logrus"
)

type CheckClusterDeregistrationStep struct {
	reconcilerClient reconciler.Client
	timeout          time.Duration
	operationManager *process.DeprovisionOperationManager
}

func NewCheckClusterDeregistrationStep(os storage.Operations, cli reconciler.Client, timeout time.Duration) *CheckClusterDeregistrationStep {
	return &CheckClusterDeregistrationStep{
		reconcilerClient: cli,
		timeout:          timeout,
		operationManager: process.NewDeprovisionOperationManager(os),
	}
}

func (s *CheckClusterDeregistrationStep) Name() string {
	return "Check_Cluster_Deregistration"
}

func (s *CheckClusterDeregistrationStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if !operation.ClusterConfigurationDeleted {
		log.Infof("Cluster deregistration has not be executed, skipping", s.timeout)
		return operation, 0, nil
	}
	if operation.ClusterConfigurationVersion == 0 {
		log.Info("ClusterConfigurationVersion is zero, skipping")
		return operation, 0, nil
	}
	if operation.TimeSinceReconcilerDeregistrationTriggered() > s.timeout {
		log.Errorf("Cluster deregistration has reached the time limit: %s", s.timeout)
		modifiedOp, d, _ := s.operationManager.UpdateOperation(operation, func(op *internal.DeprovisioningOperation) {
			op.ClusterConfigurationVersion = 0
		}, log)
		return modifiedOp, d, nil
	}

	state, err := s.reconcilerClient.GetCluster(operation.RuntimeID, operation.ClusterConfigurationVersion)
	if kebError.IsNotFoundError(err) {
		log.Info("cluster already deleted")
		modifiedOp, d, _ := s.operationManager.UpdateOperation(operation, func(op *internal.DeprovisioningOperation) {
			op.ClusterConfigurationVersion = 0
		}, log)
		return modifiedOp, d, nil
	}
	if kebError.IsTemporaryError(err) {
		log.Errorf("Reconciler GetCluster method failed (temporary error, retrying): %s", err.Error())
		return operation, 1 * time.Minute, nil
	}
	if err != nil {
		log.Errorf("Reconciler GetCluster method failed: %s", err.Error())
		modifiedOp, d, _ := s.operationManager.UpdateOperation(operation, func(op *internal.DeprovisioningOperation) {
			op.ClusterConfigurationVersion = 0
		}, log)
		return modifiedOp, d, nil
	}
	log.Debugf("Cluster configuration status %s", state.Status)

	switch state.Status {
	case reconcilerApi.StatusDeletePending, reconcilerApi.StatusDeleting, reconcilerApi.StatusDeleteErrorRetryable:
		return operation, 30 * time.Second, nil
	case reconcilerApi.StatusDeleted:
		modifiedOp, d, _ := s.operationManager.UpdateOperation(operation, func(op *internal.DeprovisioningOperation) {
			op.ClusterConfigurationVersion = 0
		}, log)
		return modifiedOp, d, nil
	case reconcilerApi.StatusDeleteError, reconcilerApi.StatusError:
		errMsg := fmt.Sprintf("Reconciler deletion failed. %v", reconciler.PrettyFailures(state))
		log.Warnf(errMsg)
		modifiedOp, d, _ := s.operationManager.UpdateOperation(operation, func(op *internal.DeprovisioningOperation) {
			op.ClusterConfigurationVersion = 0
		}, log)
		return modifiedOp, d, nil
	default:
		log.Warnf("Unexpected state: %s", state.Status)
		return operation, time.Minute, nil
	}
}
