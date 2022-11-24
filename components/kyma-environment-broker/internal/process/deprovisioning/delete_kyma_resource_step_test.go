package deprovisioning

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDeleteKymaResource_HappyFlow(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = "kyma-system"

	kcpClient := fake.NewClientBuilder().Build()
	memoryStorage := storage.NewMemoryStorage()
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDeleteKymaResourceStep(memoryStorage.Operations(), kcpClient)
	memoryStorage.Operations().InsertOperation(operation)

	// When
	_, backoff, err := step.Run(operation, logger.NewLogSpy().Logger)

	// Then
	assert.Zero(t, backoff)
}
