package deprovisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	provisionerAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestRemoveRuntimeStep_Run(t *testing.T) {
	t.Run("Should repeat process when deprovisioning call to provisioner succeeded", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()

		operation := fixture.FixDeprovisioningOperation(fixOperationID, fixInstanceID)
		operation.GlobalAccountID = fixGlobalAccountID
		operation.RuntimeID = fixRuntimeID
		err := memoryStorage.Operations().InsertDeprovisioningOperation(operation)
		assert.NoError(t, err)

		err = memoryStorage.Instances().Insert(fixInstanceRuntimeStatus())
		assert.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("DeprovisionRuntime", fixGlobalAccountID, fixRuntimeID).Return(fixProvisionerOperationID, nil)

		step := NewRemoveRuntimeStep(memoryStorage.Operations(), memoryStorage.Instances(), provisionerClient, time.Minute)

		// when
		entry := log.WithFields(logrus.Fields{"step": "TEST"})
		result, repeat, err := step.Run(operation.Operation, entry)

		// then
		assert.NoError(t, err)
		assert.Equal(t, 0*time.Second, repeat)
		assert.Equal(t, fixProvisionerOperationID, result.ProvisionerOperationID)

		instance, err := memoryStorage.Instances().GetByID(result.InstanceID)
		assert.NoError(t, err)
		assert.Equal(t, instance.RuntimeID, fixRuntimeID)

	})
}
