package provisioning

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCreateRuntimeForOwnCluster_Run(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixOperationCreateRuntime(t, broker.OwnClusterPlanID, "europe-west3")
	operation.ShootDomain = "kyma.org"
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	step := NewCreateRuntimeForOwnClusterStep(memoryStorage.Operations(), memoryStorage.Instances())

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	operation, _, err = step.Run(operation, entry)

	// then

	storedInstance, err := memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)
	assert.NotEmpty(t, storedInstance.RuntimeID)

	storedOperation, err := memoryStorage.Operations().GetOperationByID(operationID)
	assert.NoError(t, err)
	assert.Empty(t, storedOperation.ProvisionerOperationID)
	assert.Equal(t, storedInstance.RuntimeID, storedOperation.RuntimeID)
}
