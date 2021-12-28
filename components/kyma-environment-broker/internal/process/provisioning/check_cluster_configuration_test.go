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

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
)

func TestCheckClusterConfigurationStep_ClusterReady(t *testing.T) {
	st := storage.NewMemoryStorage()
	operation := fixture.FixProvisioningOperation("op-id", "inst-id")
	operation.ClusterConfigurationVersion = 1
	recClient := reconciler.NewFakeClient()
	recClient.ApplyClusterConfig(reconcilerApi.Cluster{
		RuntimeID:    operation.RuntimeID,
		RuntimeInput: reconcilerApi.RuntimeInput{},
		KymaConfig:   reconcilerApi.KymaConfig{},
		Metadata:     reconcilerApi.Metadata{},
		Kubeconfig:   "kubeconfig",
	})
	recClient.ChangeClusterState(operation.RuntimeID, 1, reconcilerApi.StatusReady)

	step := NewCheckClusterConfigurationStep(st.Operations(), recClient, time.Minute)
	st.Operations().InsertProvisioningOperation(operation)

	// when
	_, d, err := step.Run(operation, logger.NewLogSpy().Logger)

	// then
	require.NoError(t, err)
	assert.Zero(t, d)
}

func TestCheckClusterConfigurationStep_InProgress(t *testing.T) {
	for _, state := range []reconcilerApi.Status{reconcilerApi.StatusReconciling, reconcilerApi.StatusReconcilePending} {
		t.Run(fmt.Sprintf("shopuld repeat for state %s", state), func(t *testing.T) {
			st := storage.NewMemoryStorage()
			operation := fixture.FixProvisioningOperation("op-id", "inst-id")
			operation.ClusterConfigurationVersion = 1
			recClient := reconciler.NewFakeClient()
			recClient.ApplyClusterConfig(reconcilerApi.Cluster{
				RuntimeID:    operation.RuntimeID,
				RuntimeInput: reconcilerApi.RuntimeInput{},
				KymaConfig:   reconcilerApi.KymaConfig{},
				Metadata:     reconcilerApi.Metadata{},
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
	recClient.ApplyClusterConfig(reconcilerApi.Cluster{
		RuntimeID:    operation.RuntimeID,
		RuntimeInput: reconcilerApi.RuntimeInput{},
		KymaConfig:   reconcilerApi.KymaConfig{},
		Metadata:     reconcilerApi.Metadata{},
		Kubeconfig:   "kubeconfig",
	})
	recClient.ChangeClusterState(operation.RuntimeID, 1, reconcilerApi.StatusError)

	step := NewCheckClusterConfigurationStep(st.Operations(), recClient, time.Minute)
	st.Operations().InsertProvisioningOperation(operation)

	// when
	_, d, err := step.Run(operation, logger.NewLogSpy().Logger)

	// then
	require.Error(t, err)
	assert.Zero(t, d)
}
