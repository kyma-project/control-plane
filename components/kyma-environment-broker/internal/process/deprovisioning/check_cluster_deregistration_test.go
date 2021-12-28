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

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
)

func TestCheckClusterDeregistrationStep(t *testing.T) {
	for tn, tc := range map[string]struct {
		State                reconcilerApi.Status
		ExpectedZeroDuration bool
	}{
		"Deleting (pending)": {
			State:                reconcilerApi.StatusDeletePending,
			ExpectedZeroDuration: false,
		},
		"Deleting": {
			State:                reconcilerApi.StatusDeleting,
			ExpectedZeroDuration: false,
		},
		"Deleted": {
			State:                reconcilerApi.StatusDeleted,
			ExpectedZeroDuration: true,
		},
		"Delete error": {
			State:                reconcilerApi.StatusDeleteError,
			ExpectedZeroDuration: true,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			st := storage.NewMemoryStorage()
			operation := fixture.FixDeprovisioningOperation("op-id", "inst-id")
			operation.ClusterConfigurationVersion = 1
			operation.ClusterConfigurationDeleted = true
			recClient := reconciler.NewFakeClient()
			recClient.ApplyClusterConfig(reconcilerApi.Cluster{
				RuntimeID:    operation.RuntimeID,
				RuntimeInput: reconcilerApi.RuntimeInput{},
				KymaConfig:   reconcilerApi.KymaConfig{},
				Metadata:     reconcilerApi.Metadata{},
				Kubeconfig:   "kubeconfig",
			})
			recClient.ChangeClusterState(operation.RuntimeID, 1, tc.State)

			step := NewCheckClusterDeregistrationStep(recClient, time.Minute)
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
