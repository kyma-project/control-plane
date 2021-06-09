package deprovisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	provisionerAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestRemoveRuntimeStep_Run(t *testing.T) {
	t.Run("Should repeat process when deprovisioning call to provisioner succeeded", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()

		operation := fixture.FixDeprovisioningOperation(fixOperationID, fixInstanceID)
		err := memoryStorage.Operations().InsertDeprovisioningOperation(operation)
		assert.NoError(t, err)

		err = memoryStorage.Instances().Insert(fixInstanceRuntimeStatus())
		assert.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("DeprovisionRuntime", fixGlobalAccountID, fixRuntimeID).Return(fixProvisionerOperationID, nil)

		step := NewRemoveRuntimeStep(memoryStorage.Operations(), memoryStorage.Instances(), provisionerClient)

		// when
		entry := log.WithFields(logrus.Fields{"step": "TEST"})
		result, repeat, err := step.Run(operation, entry)

		// then
		assert.NoError(t, err)
		assert.Equal(t, 1*time.Second, repeat)
		assert.Equal(t, fixProvisionerOperationID, result.ProvisionerOperationID)

		instance, err := memoryStorage.Instances().GetByID(result.InstanceID)
		assert.NoError(t, err)
		assert.Equal(t, instance.RuntimeID, fixRuntimeID)

	})

	t.Run("Should mark operation as succeeded and repeat process when runtime not exist", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()

		operation := fixture.FixDeprovisioningOperation(fixOperationID, fixInstanceID)
		operation.ProvisionerOperationID = ""
		operation.RuntimeID = ""

		err := memoryStorage.Operations().InsertDeprovisioningOperation(operation)
		assert.NoError(t, err)

		fixedInstance := fixInstanceRuntimeStatus()
		fixedInstance.RuntimeID = ""
		err = memoryStorage.Instances().Insert(fixedInstance)
		assert.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("DeprovisionRuntime", fixGlobalAccountID, fixRuntimeID).Return(fixProvisionerOperationID, nil)

		step := NewRemoveRuntimeStep(memoryStorage.Operations(), memoryStorage.Instances(), provisionerClient)

		// when
		entry := log.WithFields(logrus.Fields{"step": "TEST"})
		result, repeat, err := step.Run(operation, entry)

		// then
		assert.NoError(t, err)
		assert.Equal(t, domain.Succeeded, result.State)
		assert.Equal(t, 1*time.Second, repeat)
		assert.Equal(t, "", result.ProvisionerOperationID)
		assert.Equal(t, "", result.RuntimeID)
	})
}
