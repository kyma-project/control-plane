package provisioning

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

func TestStartStep_RunIfDeprovisioningInProgress(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	step := NewStartStep(st.Operations(), st.Instances())
	dOp := fixture.FixDeprovisioningOperation("d-op-id", "instance-id")
	dOp.State = domain.InProgress
	dOp.Temporary = true
	pOp := fixture.FixProvisioningOperation("p-op-id", "instance-id")
	pOp.State = orchestration.Pending
	inst := fixture.FixInstance("instance-id")

	st.Instances().Insert(inst)
	st.Operations().InsertDeprovisioningOperation(dOp)
	st.Operations().InsertProvisioningOperation(pOp)

	// when
	operation, retry, err := step.Run(pOp, logrus.New())

	// then
	assert.Equal(t, domain.LastOperationState(orchestration.Pending), operation.State)
	assert.NoError(t, err)
	assert.NotZero(t, retry)
}

func TestStartStep_RunIfDeprovisioningDone(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	step := NewStartStep(st.Operations(), st.Instances())
	dOp := fixture.FixDeprovisioningOperation("d-op-id", "instance-id")
	dOp.State = domain.Succeeded
	dOp.Temporary = true
	pOp := fixture.FixProvisioningOperation("p-op-id", "instance-id")
	pOp.State = orchestration.Pending
	inst := fixture.FixInstance("instance-id")

	st.Instances().Insert(inst)
	st.Operations().InsertDeprovisioningOperation(dOp)
	st.Operations().InsertProvisioningOperation(pOp)

	// when
	operation, retry, err := step.Run(pOp, logrus.New())

	// then
	assert.Equal(t, domain.InProgress, operation.State)
	assert.NoError(t, err)
	assert.Zero(t, retry)
	storedOp, err := st.Operations().GetProvisioningOperationByID("p-op-id")
	assert.NoError(t, err)
	assert.Equal(t, domain.InProgress, storedOp.State)
}
