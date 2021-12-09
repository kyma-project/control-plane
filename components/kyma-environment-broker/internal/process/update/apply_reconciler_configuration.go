package update

import (
	"fmt"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type ApplyReconcilerConfigurationStep struct {
	reconcilerClient    reconciler.Client
	operationManager    *process.UpdateOperationManager
	runtimeStateStorage storage.RuntimeStates
}

func NewApplyReconcilerConfigurationStep(os storage.Operations, runtimeStorage storage.RuntimeStates, reconcilerClient reconciler.Client) *ApplyReconcilerConfigurationStep {
	return &ApplyReconcilerConfigurationStep{
		reconcilerClient:    reconcilerClient,
		operationManager:    process.NewUpdateOperationManager(os),
		runtimeStateStorage: runtimeStorage,
	}
}

func (s *ApplyReconcilerConfigurationStep) Name() string {
	return "Apply_Reconciler_Configuration"
}

func (s *ApplyReconcilerConfigurationStep) Run(operation internal.UpdatingOperation, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	cluster := operation.LastRuntimeState.ClusterSetup
	if err := s.runtimeStateStorage.Insert(internal.NewRuntimeStateWithReconcilerInput(cluster.Cluster, operation.ID, cluster)); err != nil {
		log.Errorf("cannot insert runtimeState with reconciler payload: %s", err)
		return operation, 10 * time.Second, nil
	}

	log.Infof("Applying Cluster Configuration: cluster(runtimeID)=%s, kymaVersion=%s, kymaProfile=%s, components=[%s]",
		cluster.Cluster, cluster.KymaConfig.Version, cluster.KymaConfig.Profile, s.componentList(*cluster))
	state, err := s.reconcilerClient.ApplyClusterConfig(*cluster)
	switch {
	case kebError.IsTemporaryError(err):
		msg := fmt.Sprintf("Request to Reconciler failed: %s", err.Error())
		log.Error(msg)
		return operation, 5 * time.Second, nil
	case err != nil:
		msg := fmt.Sprintf("Request to Reconciler failed: %s", err.Error())
		log.Error(msg)
		return operation, 0, err
	}

	log.Infof("Reconciler configuration version %d", state.ConfigurationVersion)
	updatedOperation, repeat := s.operationManager.UpdateOperation(operation, func(op *internal.UpdatingOperation) {
		op.ClusterConfigurationVersion = state.ConfigurationVersion
	}, log)
	if repeat != 0 {
		log.Errorf("cannot save cluster configuration version")
		return operation, repeat, nil
	}
	return updatedOperation, 0, nil
}

func (s *ApplyReconcilerConfigurationStep) componentList(cluster reconciler.Cluster) string {
	vals := []string{}
	for _, c := range cluster.KymaConfig.Components {
		vals = append(vals, c.Component)
	}
	return strings.Join(vals, ", ")
}
