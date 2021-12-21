package upgrade_kyma

import (
	"fmt"
	"time"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

// CheckClusterConfigurationStep checks if the SKR configuration is applied (by reconciler)
type CheckClusterConfigurationStep struct {
	reconcilerClient      reconciler.Client
	operationManager      *process.UpgradeKymaOperationManager
	reconciliationTimeout time.Duration
}

func NewCheckClusterConfigurationStep(os storage.Operations,
	reconcilerClient reconciler.Client,
	provisioningTimeout time.Duration) *CheckClusterConfigurationStep {
	return &CheckClusterConfigurationStep{
		reconcilerClient:      reconcilerClient,
		operationManager:      process.NewUpgradeKymaOperationManager(os),
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
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", s.reconciliationTimeout), log)
	}

	if operation.ClusterConfigurationVersion == 0 {
		// upgrade was trigerred in reconciler, no need to call provisioner and create UpgradeRuntimeInput
		// TODO: deal with skipping steps in case of calling reconciler for Kyma 2.0 upgrade - introduce stages
		log.Debugf("Cluster configuration not yet created, skipping")
		return operation, 0, nil
	}

	state, err := s.reconcilerClient.GetCluster(operation.InstanceDetails.RuntimeID, operation.ClusterConfigurationVersion)
	if kebError.IsTemporaryError(err) {
		log.Errorf("Reconciler GetCluster method failed (temporary error, retrying): %s", err.Error())
		return operation, 1 * time.Minute, nil
	}
	if err != nil {
		log.Errorf("Reconciler GetCluster method failed: %s", err.Error())
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("unable to get cluster state: %s", err.Error()), log)
	}
	log.Debugf("Cluster configuration status %s", state.Status)

	switch state.Status {
	case reconciler.ClusterStatusReconciling, reconciler.ClusterStatusPending:
		return operation, 30 * time.Second, nil
	case reconciler.ClusterStatusReady:
		return s.operationManager.OperationSucceeded(operation, "Cluster configuration ready", log)
	case reconciler.ClusterStatusError:
		errMsg := fmt.Sprintf("Reconciler failed. %v", state.PrettyFailures())
		log.Warnf(errMsg)
		return s.operationManager.OperationFailed(operation, errMsg, log)
	default:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("unknown cluster status: %s", state.Status), log)
	}
}
