package deprovisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestRemoveInstanceStep_HappyPathForPermanentRemoval(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixture.FixDeprovisioningOperationAsOperation(operationID, instanceID)
	instance := fixture.FixInstance(instanceID)

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewRemoveInstanceStep(memoryStorage.Instances(), memoryStorage.Operations())

	// when
	operation, repeat, err := step.Run(operation, log)

	assert.NoError(t, err)

	// then
	operationFromStorage, err := memoryStorage.Operations().GetOperationByID(operationID)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(operationFromStorage.ProvisioningParameters.ErsContext.UserID))

	_, err = memoryStorage.Instances().GetByID(instanceID)
	assert.ErrorContains(t, err, "not exist")

	assert.Equal(t, time.Duration(0), repeat)
	assert.Equal(t, domain.Succeeded, operation.State)
}

func TestRemoveInstanceStep_HappyPathForSuspension(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixture.FixSuspensionOperationAsOperation(operationID, instanceID)
	instance := fixture.FixInstance(instanceID)

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewRemoveInstanceStep(memoryStorage.Instances(), memoryStorage.Operations())

	// when
	operation, repeat, err := step.Run(operation, log)

	assert.NoError(t, err)

	// then
	operationFromStorage, err := memoryStorage.Operations().GetOperationByID(operationID)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(operationFromStorage.RuntimeID))

	instanceFromStorage, err := memoryStorage.Instances().GetByID(instanceID)
	assert.Equal(t, 0, len(instanceFromStorage.RuntimeID))

	assert.Equal(t, time.Duration(0), repeat)
	assert.Equal(t, domain.Succeeded, operation.State)
}
