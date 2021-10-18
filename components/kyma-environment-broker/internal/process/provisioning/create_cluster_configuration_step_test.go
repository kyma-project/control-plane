package provisioning

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateClusterConfigurationStep_Run(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	reconcilerClient := reconciler.NewFakeClient()
	step := NewCreateClusterConfiguration(st.Operations(), st.RuntimeStates(), reconcilerClient)
	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.RuntimeID = runtimeID
	runtimeStateID := "runtimeState1"
	runtimeState := fixture.FixRuntimeState(runtimeStateID, runtimeID, operationID)
	st.Operations().InsertProvisioningOperation(operation)
	st.RuntimeStates().Insert(runtimeState)

	// when
	_, d, err := step.Run(operation, logrus.New())

	// then
	require.NoError(t, err)
	assert.Zero(t, d)

	// when
	gotRuntimeState, err := st.RuntimeStates().GetLastByRuntimeID(runtimeID)

	// then
	require.NoError(t, err)
	assert.Equal(t, runtimeID, gotRuntimeState.RuntimeID)
}
