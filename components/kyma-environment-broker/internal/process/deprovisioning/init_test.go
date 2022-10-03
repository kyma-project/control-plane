package deprovisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	fixOperationID            = "17f3ddba-1132-466d-a3c5-920f544d7ea6"
	fixInstanceID             = "9d75a545-2e1e-4786-abd8-a37b14e185b9"
	fixRuntimeID              = "ef4e3210-652c-453e-8015-bba1c1cd1e1c"
	fixGlobalAccountID        = "abf73c71-a653-4951-b9c2-a26d6c2cccbd"
	fixProvisionerOperationID = "e04de524-53b3-4890-b05a-296be393e4ba"
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
	s.Operations().InsertOperation(pOp)
}

func prepareDeprovisioningOperation(s storage.BrokerStorage, state domain.LastOperationState) internal.Operation {
	dOperation := fixture.FixDeprovisioningOperation("dop-id", instanceID)
	dOperation.State = state
	s.Operations().InsertOperation(dOperation.Operation)
	return dOperation.Operation
}

func fixDeprovisioningOperation() internal.DeprovisioningOperation {
	deprovisioniningOperation := fixture.FixDeprovisioningOperation(fixOperationID, fixInstanceID)
	return deprovisioniningOperation
}

func fixProvisioningOperation() internal.Operation {
	provisioningOperation := fixture.FixProvisioningOperation(fixOperationID, fixInstanceID)
	return provisioningOperation
}

func fixInstanceRuntimeStatus() internal.Instance {
	instance := fixture.FixInstance(fixInstanceID)
	instance.RuntimeID = fixRuntimeID
	instance.GlobalAccountID = fixGlobalAccountID

	return instance
}
