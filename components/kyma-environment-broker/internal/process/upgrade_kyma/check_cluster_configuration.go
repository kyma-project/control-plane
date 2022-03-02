package upgrade_kyma

import (
	"fmt"
	"time"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

// CheckClusterConfigurationStep checks if the SKR configuration is applied (by reconciler)
type CheckClusterConfigurationStep struct {
	reconcilerClient      reconciler.Client
	operationManager      *process.UpgradeKymaOperationManager
	evaluationManager     *avs.EvaluationManager
	reconciliationTimeout time.Duration
}

func NewCheckClusterConfigurationStep(os storage.Operations,
	reconcilerClient reconciler.Client,
	evaluationManager *avs.EvaluationManager,
	provisioningTimeout time.Duration) *CheckClusterConfigurationStep {
	return &CheckClusterConfigurationStep{
		reconcilerClient:      reconcilerClient,
		operationManager:      process.NewUpgradeKymaOperationManager(os),
		evaluationManager:     evaluationManager,
		reconciliationTimeout: provisioningTimeout,
	}
}

var _ Step = (*CheckClusterConfigurationStep)(nil)

func (s *CheckClusterConfigurationStep) Name() string {
	return "Check_Cluster_Configuration"
}

func (s *CheckClusterConfigurationStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > s.reconciliationTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.restoreAvsFailOperation(operation, fmt.Sprintf("operation has reached the time limit: %s", s.reconciliationTimeout), log)
	}

	if operation.ClusterConfigurationVersion == 0 || !operation.ClusterConfigurationApplied {
		// upgrade was trigerred in reconciler, no need to call provisioner and create UpgradeRuntimeInput
		// TODO: deal with skipping steps in case of calling reconciler for Kyma 2.0 upgrade - introduce stages
		log.Infof("Cluster configuration not yet created, skipping")
		return operation, 0, nil
	}

	state, err := s.reconcilerClient.GetCluster(operation.InstanceDetails.RuntimeID, operation.ClusterConfigurationVersion)
	if kebError.IsTemporaryError(err) {
		log.Errorf("Reconciler GetCluster method failed (temporary error, retrying): %s", err.Error())
		return operation, 1 * time.Minute, nil
	}
	if err != nil {
		log.Errorf("Reconciler GetCluster method failed: %s", err.Error())
		return s.restoreAvsFailOperation(operation, fmt.Sprintf("unable to get cluster state: %s", err.Error()), log)
	}
	log.Debugf("Cluster configuration status %s", state.Status)

	// Ensure AVS evaluations status:
	//  - set to maintenance in reconcile_pending, reconciling, error_retryable
	//  - restore when reconciler status is ready or unknown/failure (operation terminal)
	switch state.Status {
	case reconcilerApi.StatusReconciling, reconcilerApi.StatusReconcilePending, reconcilerApi.StatusReconcileErrorRetryable:
		operation, err = SetAvsStatusMaintenance(s.evaluationManager, s.operationManager, operation, log)
	default:
		operation, err = RestoreAvsStatus(s.evaluationManager, s.operationManager, operation, log)
	}
	if err != nil {
		if kebError.IsTemporaryError(err) {
			return operation, 30 * time.Second, nil
		}
		return s.operationManager.OperationFailed(operation, err.Error(), nil, log)
	}

	switch state.Status {
	case reconcilerApi.StatusReconciling, reconcilerApi.StatusReconcilePending:
		return operation, 30 * time.Second, nil
	case reconcilerApi.StatusReconcileErrorRetryable:
		log.Infof("Reconciler failed with retryable, rechecking in 10 minutes.")
		return operation, 10 * time.Minute, nil
	case reconcilerApi.StatusReady:
		return s.operationManager.OperationSucceeded(operation, "Cluster configuration ready", log)
	case reconcilerApi.StatusError:
		errMsg := fmt.Sprintf("Reconciler failed. %v", reconciler.PrettyFailures(state))
		log.Warnf(errMsg)
		return s.operationManager.OperationFailed(operation, errMsg, nil, log)
	default:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("unknown cluster status: %s", state.Status), nil, log)
	}
}

func (s *CheckClusterConfigurationStep) restoreAvsFailOperation(operation internal.UpgradeKymaOperation, description string, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	operation, err := RestoreAvsStatus(s.evaluationManager, s.operationManager, operation, log)
	if kebError.IsTemporaryError(err) {
		return operation, 30 * time.Second, nil
	}
	return s.operationManager.OperationFailed(operation, description, err, log)
}
