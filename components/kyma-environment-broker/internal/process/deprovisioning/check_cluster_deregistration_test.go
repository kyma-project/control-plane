package deprovisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckClusterDeregistrationStep(t *testing.T) {
	for tn, tc := range map[string]struct {
		State                string
		ExpectedZeroDuration bool
	}{
		"Deleting (pending)": {
			State:                reconciler.ClusterStatusDeletePending,
			ExpectedZeroDuration: false,
		},
		"Deleting": {
			State:                reconciler.ClusterStatusDeleting,
			ExpectedZeroDuration: false,
		},
		"Deleted": {
			State:                reconciler.ClusterStatusDeleted,
			ExpectedZeroDuration: true,
		},
		"Delete error": {
			State:                reconciler.ClusterStatusDeleteError,
			ExpectedZeroDuration: true,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			st := storage.NewMemoryStorage()
			operation := fixture.FixDeprovisioningOperation("op-id", "inst-id")
			operation.ClusterConfigurationVersion = 1
			recClient := reconciler.NewFakeClient()
			recClient.ApplyClusterConfig(reconciler.Cluster{
				Cluster:      operation.RuntimeID,
				RuntimeInput: reconciler.RuntimeInput{},
				KymaConfig:   reconciler.KymaConfig{},
				Metadata:     reconciler.Metadata{},
				Kubeconfig:   "kubeconfig",
			})
			recClient.ChangeClusterState(operation.RuntimeID, 1, tc.State)

			step := NewCheckClusterDeregistrationStep(st.Operations(), recClient, time.Minute)
			st.Operations().InsertDeprovisioningOperation(operation)

			// when
			_, d, err := step.Run(operation, logger.NewLogSpy().Logger)

			// then
			require.NoError(t, err)
			if tc.ExpectedZeroDuration {
				assert.Zero(t, d)
			} else {
				assert.NotZero(t, d)
			}

		})
	}

}
