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

func TestRemoveInstanceStep_HappyPath(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixture.FixDeprovisioningOperationAsOperation(operationID, instanceID)
	instance := fixGCPInstance(operation.InstanceID)

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	step := NewRemoveInstanceStep(memoryStorage.Instances(), nil)

	// when
	operation, repeat, err := step.Run(operation, log)

	assert.NoError(t, err)

	// then
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
	assert.Equal(t, domain.Succeeded, operation.State)
}
