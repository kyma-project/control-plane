package upgrade_kyma

import (
	"fmt"
	"strings"
	"time"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type ApplyClusterConfigurationStep struct {
	operationManager    *process.UpgradeKymaOperationManager
	reconcilerClient    reconciler.Client
	runtimeStateStorage storage.RuntimeStates
}

func NewApplyClusterConfigurationStep(os storage.Operations, rs storage.RuntimeStates, reconcilerClient reconciler.Client) *ApplyClusterConfigurationStep {
	return &ApplyClusterConfigurationStep{
		operationManager:    process.NewUpgradeKymaOperationManager(os),
		reconcilerClient:    reconcilerClient,
		runtimeStateStorage: rs,
	}
}

func (s *ApplyClusterConfigurationStep) Name() string {
	return "Apply_Cluster_Configuration"
}

func checkBTPCredsValid(clusterConfiguration reconcilerApi.Cluster) error {
	vals := make(map[string]bool)
	requiredKeys := []string{internal.BTPOperatorClientID, internal.BTPOperatorClientSecret, internal.BTPOperatorURL, internal.BTPOperatorTokenURL}
	hasBTPOperator := false
	var errs []string
	for _, c := range clusterConfiguration.KymaConfig.Components {
		if c.Component == internal.BTPOperatorComponentName {
			hasBTPOperator = true
			for _, cfg := range c.Configuration {
				for _, key := range requiredKeys {
					if cfg.Key == key {
						vals[key] = true
						if cfg.Value == nil {
							errs = append(errs, fmt.Sprintf("missing required value for %v", key))
						}
						if val, ok := cfg.Value.(string); !ok || val == "" {
							errs = append(errs, fmt.Sprintf("missing required value for %v", key))
						}
					}
				}
			}
		}
	}
	if hasBTPOperator {
		for _, key := range requiredKeys {
			if !vals[key] {
				errs = append(errs, fmt.Sprintf("missing required key %v", key))
			}
		}
		if len(errs) != 0 {
			return fmt.Errorf("BTP Operator is about to be installed but is missing required configuration: %v", strings.Join(errs, ", "))
		}
	}
	return nil
}

func (s *ApplyClusterConfigurationStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if operation.ClusterConfigurationApplied {
		log.Infof("Cluster configuration already applied")
		return operation, 0, nil
	}
	operation.InputCreator.DisableOptionalComponent(internal.SCMigrationComponentName)
	operation.InputCreator.SetRuntimeID(operation.InstanceDetails.RuntimeID).
		SetInstanceID(operation.InstanceID).
		SetShootName(operation.InstanceDetails.ShootName).
		SetShootDomain(operation.ShootDomain).
		SetProvisioningParameters(operation.ProvisioningParameters)

	clusterConfiguration, err := operation.InputCreator.CreateClusterConfiguration()
	if err != nil {
		log.Errorf("Unable to apply cluster configuration: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "invalid operation data - cannot create cluster configuration", err, log)
	}

	if err := checkBTPCredsValid(clusterConfiguration); err != nil {
		log.Errorf("Sanity check for BTP operator configuration failed: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "invalid BTP Operator configuration", log)
	}

	err = s.runtimeStateStorage.Insert(
		internal.NewRuntimeStateWithReconcilerInput(clusterConfiguration.RuntimeID, operation.Operation.ID, &clusterConfiguration))
	if err != nil {
		log.Errorf("cannot insert runtimeState with reconciler payload: %s", err)
		return operation, 10 * time.Second, nil
	}

	log.Infof("Apply Cluster Configuration: cluster(runtimeID)=%s, kymaVersion=%s, kymaProfile=%s, components=[%s]",
		clusterConfiguration.RuntimeID,
		clusterConfiguration.KymaConfig.Version,
		clusterConfiguration.KymaConfig.Profile,
		s.componentList(clusterConfiguration))
	state, err := s.reconcilerClient.ApplyClusterConfig(clusterConfiguration)
	switch {
	case kebError.IsTemporaryError(err):
		msg := fmt.Sprintf("Request to Reconciler failed: %s", err.Error())
		log.Error(msg)
		return operation, 5 * time.Second, nil
	case err != nil:
		msg := fmt.Sprintf("Request to Reconciler failed: %s", err.Error())
		log.Error(msg)
		return s.operationManager.OperationFailed(operation, "Request to Reconciler failed", err, log)
	}
	log.Infof("Cluster configuration version %d", state.ConfigurationVersion)

	updatedOperation, repeat, _ := s.operationManager.UpdateOperation(operation, func(operation *internal.UpgradeKymaOperation) {
		operation.ClusterConfigurationVersion = state.ConfigurationVersion
		operation.ClusterConfigurationApplied = true
	}, log)
	if repeat != 0 {
		log.Errorf("cannot save cluster configuration version")
		return operation, 5 * time.Second, nil
	}

	// return some retry value to get back to initialisation step
	return updatedOperation, 5 * time.Second, nil

}

func (s *ApplyClusterConfigurationStep) componentList(cluster reconcilerApi.Cluster) string {
	vals := []string{}
	for _, c := range cluster.KymaConfig.Components {
		vals = append(vals, c.Component)
	}
	return strings.Join(vals, ", ")
}
