package upgrade_kyma

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGetKubeconfigStep(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	provisionerClient := provisioner.NewFakeClient()
	op := fixOperation("op-id")
	op.Kubeconfig = ""

	input, err := op.InputCreator.CreateProvisionRuntimeInput()
	require.NoError(t, err)
	provisionerClient.ProvisionRuntimeWithIDs(op.GlobalAccountID, op.SubAccountID, op.RuntimeID, op.ID, input)

	step := NewGetKubeconfigStep(st.Operations(), provisionerClient)
	st.Operations().InsertUpgradeKymaOperation(op)

	// when
	newOp, d, err := step.Run(op, logrus.New())

	// then
	require.NoError(t, err)
	assert.Zero(t, d)
	assert.NotEmpty(t, newOp.Kubeconfig)
}
