package provisioning

import (
	"fmt"
	"time"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

// CheckClusterConfigurationStep checks if the SKR configuration is applied (by reconciler)
type CheckClusterConfigurationStep struct {
	reconcilerClient    reconciler.Client
	operationManager    *process.ProvisionOperationManager
	provisioningTimeout time.Duration
}

func NewCheckClusterConfigurationStep(os storage.Operations,
	reconcilerClient reconciler.Client,
	provisioningTimeout time.Duration) *CheckClusterConfigurationStep {
	return &CheckClusterConfigurationStep{
		reconcilerClient:    reconcilerClient,
		operationManager:    process.NewProvisionOperationManager(os),
		provisioningTimeout: provisioningTimeout,
	}
}

var _ Step = (*CheckClusterConfigurationStep)(nil)

func (s *CheckClusterConfigurationStep) Name() string {
	return "Check_Cluster_Configuration"
}

func (s *CheckClusterConfigurationStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > s.provisioningTimeout {
		return s.handleTimeout(operation, log)
	}

	state, err := s.reconcilerClient.GetCluster(operation.RuntimeID, operation.ClusterConfigurationVersion)
	if kebError.IsTemporaryError(err) {
		log.Errorf("Reconciler GetCluster method failed (temporary error, retrying): %s", err.Error())
		return operation, 1 * time.Minute, nil
	}
	if err != nil {
		log.Errorf("Reconciler GetCluster method failed: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "unable to get cluster state", err, log)
	}
	log.Debugf("Cluster configuration status %s", state.Status)

	switch state.Status {
	case reconcilerApi.StatusReconciling, reconcilerApi.StatusReconcilePending:
		return operation, 30 * time.Second, nil
	case reconcilerApi.StatusReconcileErrorRetryable:
		log.Infof("Reconciler failed with retryable, rechecking in 10 minutes.")
		return operation, 10 * time.Minute, nil
	case reconcilerApi.StatusReady:
		return operation, 0, nil

	case reconcilerApi.StatusError:
		errMsg := fmt.Sprintf("Reconciler failed. %v", reconciler.PrettyFailures(state))
		log.Warnf(errMsg)
		return s.operationManager.OperationFailed(operation, "Reconciler failed with error cluster status", reconciler.NewReconcilerError(state.Failures, errMsg), log)
	default:
		errMsg := fmt.Sprintf("unknown cluster status: %s", state.Status)
		return s.operationManager.OperationFailed(operation, "Reconciler failed with unknown cluster status", reconciler.NewReconcilerError(state.Failures, errMsg), log)
	}
}

func (s *CheckClusterConfigurationStep) handleTimeout(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	log.Warnf("Operation has reached the time limit (%v): updated operation time: %s", s.provisioningTimeout, operation.UpdatedAt)
	log.Infof("Deleting cluster %s", operation.RuntimeID)
	/*
		If the reconciliation timeouted, we have to delete cluster.
		In case of an error, try few times.
	*/
	err := wait.PollImmediate(5*time.Second, 30*time.Second, func() (bool, error) {
		err := s.reconcilerClient.DeleteCluster(operation.RuntimeID)
		if err != nil {
			log.Warnf("Unable to delete cluster: %s", err.Error())
		}
		return err == nil, nil
	})
	if err != nil {
		log.Errorf("Unable to delete cluster: %s", err.Error())
	}
	return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", s.provisioningTimeout), err, log)
}
