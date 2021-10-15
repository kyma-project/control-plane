package provisioning

import (
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckClusterConfigurationStep_ClusterReady(t *testing.T) {
	st := storage.NewMemoryStorage()
	operation := fixture.FixProvisioningOperation("op-id", "inst-id")
	operation.ClusterConfigurationVersion = 1
	recClient := reconciler.NewFakeClient()
	recClient.ApplyClusterConfig(reconciler.Cluster{
		Cluster:      operation.RuntimeID,
		RuntimeInput: reconciler.RuntimeInput{},
		KymaConfig:   reconciler.KymaConfig{},
		Metadata:     reconciler.Metadata{},
		Kubeconfig:   "kubeconfig",
	})
	recClient.ChangeClusterState(operation.RuntimeID, 1, reconciler.ClusterStatusReady)

	step := NewCheckClusterConfigurationStep(st.Operations(), recClient, time.Minute)
	st.Operations().InsertProvisioningOperation(operation)

	// when
	_, d, err := step.Run(operation, logger.NewLogSpy().Logger)

	// then
	require.NoError(t, err)
	assert.Zero(t, d)
}

func TestCheckClusterConfigurationStep_InProgress(t *testing.T) {
	for _, state := range []string{reconciler.ClusterStatusReconciling, reconciler.ClusterStatusPending} {
		t.Run(fmt.Sprintf("shopuld repeat for state %s", state), func(t *testing.T) {
			st := storage.NewMemoryStorage()
			operation := fixture.FixProvisioningOperation("op-id", "inst-id")
			operation.ClusterConfigurationVersion = 1
			recClient := reconciler.NewFakeClient()
			recClient.ApplyClusterConfig(reconciler.Cluster{
				Cluster:      operation.RuntimeID,
				RuntimeInput: reconciler.RuntimeInput{},
				KymaConfig:   reconciler.KymaConfig{},
				Metadata:     reconciler.Metadata{},
				Kubeconfig:   "kubeconfig",
			})
			recClient.ChangeClusterState(operation.RuntimeID, 1, state)

			step := NewCheckClusterConfigurationStep(st.Operations(), recClient, time.Minute)
			st.Operations().InsertProvisioningOperation(operation)

			// when
			_, d, err := step.Run(operation, logger.NewLogSpy().Logger)

			// then
			require.NoError(t, err)
			assert.True(t, d > 0)
		})
	}
}

func TestCheckClusterConfigurationStep_ClusterFailed(t *testing.T) {
	st := storage.NewMemoryStorage()
	operation := fixture.FixProvisioningOperation("op-id", "inst-id")
	operation.ClusterConfigurationVersion = 1
	recClient := reconciler.NewFakeClient()
	recClient.ApplyClusterConfig(reconciler.Cluster{
		Cluster:      operation.RuntimeID,
		RuntimeInput: reconciler.RuntimeInput{},
		KymaConfig:   reconciler.KymaConfig{},
		Metadata:     reconciler.Metadata{},
		Kubeconfig:   "kubeconfig",
	})
	recClient.ChangeClusterState(operation.RuntimeID, 1, reconciler.ClusterStatusError)

	step := NewCheckClusterConfigurationStep(st.Operations(), recClient, time.Minute)
	st.Operations().InsertProvisioningOperation(operation)

	// when
	_, d, err := step.Run(operation, logger.NewLogSpy().Logger)

	// then
	require.Error(t, err)
	assert.Zero(t, d)
}
