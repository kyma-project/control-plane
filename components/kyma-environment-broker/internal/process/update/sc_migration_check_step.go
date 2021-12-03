package update

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/sirupsen/logrus"
)

type CheckReconcilerState struct {
	reconcilerClient reconciler.Client
}

func NewCheckReconcilerState(reconcilerClient reconciler.Client) *CheckReconcilerState {
	return &CheckReconcilerState{
		reconcilerClient: reconcilerClient,
	}
}

func (s *CheckReconcilerState) Name() string {
	return "SCMigrationCheck"
}

func (s *CheckReconcilerState) Run(operation internal.UpdatingOperation, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
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
