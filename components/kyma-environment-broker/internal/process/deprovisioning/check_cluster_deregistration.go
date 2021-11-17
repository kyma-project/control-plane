package deprovisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/sirupsen/logrus"
)

type CheckClusterDeregistrationStep struct {
	reconcilerClient reconciler.Client
	timeout          time.Duration
}

func NewCheckClusterDeregistrationStep(cli reconciler.Client, timeout time.Duration) *CheckClusterDeregistrationStep {
	return &CheckClusterDeregistrationStep{
		reconcilerClient: cli,
		timeout:          timeout,
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
	if time.Since(operation.UpdatedAt) > s.timeout {
		log.Infof("Cluster deregistration has reached the time limit: %s", s.timeout)
		return operation, 0, nil
	}

	state, err := s.reconcilerClient.GetCluster(operation.RuntimeID, operation.ClusterConfigurationVersion)
	if kebError.IsNotFoundError(err) {
		log.Info("cluster already deleted")
		return operation, 0, nil
	}
	if kebError.IsTemporaryError(err) {
		log.Errorf("Reconciler GetCluster method failed (temporary error, retrying): %s", err.Error())
		return operation, 1 * time.Minute, nil
	}
	if err != nil {
		log.Errorf("Reconciler GetCluster method failed: %s", err.Error())
		return operation, 0, nil
	}
	log.Debugf("Cluster configuration status %s", state.Status)

	switch state.Status {
	case reconciler.ClusterStatusDeletePending, reconciler.ClusterStatusDeleting:
		return operation, 30 * time.Second, nil
	case reconciler.ClusterStatusDeleted:
		return operation, 0, nil
	case reconciler.ClusterStatusDeleteError:
		errMsg := fmt.Sprintf("Reconciler deletion failed. %v", state.PrettyFailures())
		log.Warnf(errMsg)
		return operation, 0, nil
	default:
		log.Warnf("Unexpected state: %s", state.Status)
		return operation, time.Minute, nil
	}
}
