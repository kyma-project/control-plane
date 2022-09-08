package deprovisioning

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestInitStep_happyPath(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()
	prepareProvisionedInstance(memoryStorage)
	dOp := prepareDeprovisioningOperation(memoryStorage, orchestration.Pending)

	svc := NewInitStep(memoryStorage.Operations(), memoryStorage.Instances(), 90*time.Second)

	// when
	op, d, err := svc.Run(dOp, log)

	// then
	assert.Equal(t, domain.InProgress, op.State)
	assert.NoError(t, err)
	assert.Zero(t, d)
	dbOp, _ := memoryStorage.Operations().GetOperationByID(op.ID)
	assert.Equal(t, domain.InProgress, dbOp.State)
}

func TestInitStep_existingUpdatingOperation(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()
	prepareProvisionedInstance(memoryStorage)
	uOp := fixture.FixUpdatingOperation("uop-id", instanceID)
	uOp.State = domain.InProgress
	memoryStorage.Operations().InsertOperation(uOp.Operation)
	dOp := prepareDeprovisioningOperation(memoryStorage, orchestration.Pending)

	svc := NewInitStep(memoryStorage.Operations(), memoryStorage.Instances(), 90*time.Second)

	// when
	op, d, err := svc.Run(dOp, log)

	// then
	assert.Equal(t, orchestration.Pending, string(op.State))
	assert.NoError(t, err)
	assert.NotZero(t, d)
	dbOp, _ := memoryStorage.Operations().GetOperationByID(op.ID)
	assert.Equal(t, orchestration.Pending, string(dbOp.State))
}

func prepareProvisionedInstance(s storage.BrokerStorage) {
	inst := fixture.FixInstance(instanceID)
	s.Instances().Insert(inst)
	pOp := fixture.FixProvisioningOperation("pop-id", instanceID)
	s.Operations().InsertOperation(pOp.Operation)
}

func prepareDeprovisioningOperation(s storage.BrokerStorage, state domain.LastOperationState) internal.Operation {
	dOperation := fixture.FixDeprovisioningOperation("dop-id", instanceID)
	dOperation.State = state
	s.Operations().InsertOperation(dOperation.Operation)
	return dOperation.Operation
}
