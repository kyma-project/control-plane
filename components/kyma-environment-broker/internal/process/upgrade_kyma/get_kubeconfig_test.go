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
	step := NewGetKubeconfigStep(st.Operations(), provisionerClient)
	op := fixOperation("op-id")
	op.Kubeconfig = ""
	st.Operations().InsertUpgradeKymaOperation(op)

	// when
	newOp, d, err := step.Run(op, logrus.New())

	// then
	require.NoError(t, err)
	assert.Zero(t, d)
	assert.NotEmpty(t, newOp.Kubeconfig)
}
