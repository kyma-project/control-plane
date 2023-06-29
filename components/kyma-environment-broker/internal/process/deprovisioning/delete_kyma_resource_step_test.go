package deprovisioning

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const kymaTemplate = `
apiVersion: operator.kyma-project.io/v1beta2
kind: Kyma
metadata:
  name: my-kyma
  namespace: kyma-system
spec:
  sync:
    strategy: secret
  channel: stable
  modules: []
`

func TestDeleteKymaResource_HappyFlow(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = "kyma-system"

	kcpClient := fake.NewClientBuilder().Build()
	memoryStorage := storage.NewMemoryStorage()
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDeleteKymaResourceStep(memoryStorage.Operations(), kcpClient, fakeConfigProvider{}, "2.0")
	memoryStorage.Operations().InsertOperation(operation)

	// When
	_, backoff, err := step.Run(operation, logger.NewLogSpy().Logger)

	// Then
	assert.Zero(t, backoff)
}

func TestDeleteKymaResource_EmptyRuntimeIDAndKymaResourceName(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = "kyma-system"
	operation.RuntimeID = ""
	operation.KymaResourceName = ""

	kcpClient := fake.NewClientBuilder().Build()
	memoryStorage := storage.NewMemoryStorage()
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDeleteKymaResourceStep(memoryStorage.Operations(), kcpClient, fakeConfigProvider{}, "2.0")
	memoryStorage.Operations().InsertOperation(operation)

	// When
	_, backoff, err := step.Run(operation, logger.NewLogSpy().Logger)

	// Then
	assert.Zero(t, backoff)
}

type fakeConfigProvider struct {
}

func (fakeConfigProvider) ProvideForGivenVersionAndPlan(_, _ string) (*internal.ConfigForPlan, error) {
	return &internal.ConfigForPlan{
		KymaTemplate: kymaTemplate,
	}, nil
}
