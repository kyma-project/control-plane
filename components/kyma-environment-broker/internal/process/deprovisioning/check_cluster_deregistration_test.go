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

	contract "github.com/kyma-incubator/reconciler/pkg/keb"
)

func TestCheckClusterDeregistrationStep(t *testing.T) {
	for tn, tc := range map[string]struct {
		State                contract.Status
		ExpectedZeroDuration bool
	}{
		"Deleting (pending)": {
			State:                contract.StatusDeletePending,
			ExpectedZeroDuration: false,
		},
		"Deleting": {
			State:                contract.StatusDeleting,
			ExpectedZeroDuration: false,
		},
		"Deleted": {
			State:                contract.StatusDeleted,
			ExpectedZeroDuration: true,
		},
		"Delete error": {
			State:                contract.StatusDeleteError,
			ExpectedZeroDuration: true,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			st := storage.NewMemoryStorage()
			operation := fixture.FixDeprovisioningOperation("op-id", "inst-id")
			operation.ClusterConfigurationVersion = 1
			operation.ClusterConfigurationDeleted = true
			recClient := reconciler.NewFakeClient()
			recClient.ApplyClusterConfig(contract.Cluster{
				RuntimeID:    operation.RuntimeID,
				RuntimeInput: contract.RuntimeInput{},
				KymaConfig:   contract.KymaConfig{},
				Metadata:     contract.Metadata{},
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
