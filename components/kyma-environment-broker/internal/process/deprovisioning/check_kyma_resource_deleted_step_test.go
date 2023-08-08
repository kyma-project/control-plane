package deprovisioning

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/steps"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCheckKymaResourceDeleted_HappyFlow(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = "kyma-system"
	operation.KymaTemplate = kymaTemplate

	kcpClient := fake.NewClientBuilder().Build()

	err := fixKymaResourceWithGivenRuntimeID(kcpClient, "kyma-system", "some-other-Runtime-ID")
	require.NoError(t, err)

	memoryStorage := storage.NewMemoryStorage()
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewCheckKymaResourceDeletedStep(memoryStorage.Operations(), kcpClient)

	// When
	_, backoff, err := step.Run(operation, logger.NewLogSpy().Logger)

	// Then
	assert.Zero(t, backoff)
	assertNoKymaResourceWithGivenRuntimeID(t, kcpClient, operation.KymaResourceNamespace, steps.KymaName(operation))
}

func TestCheckKymaResourceDeleted_EmptyKymaResourceName(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = "kyma-system"
	operation.RuntimeID = ""
	operation.KymaResourceName = ""
	operation.KymaTemplate = kymaTemplate

	kcpClient := fake.NewClientBuilder().Build()

	err := fixKymaResourceWithGivenRuntimeID(kcpClient, "kyma-system", "some-other-Runtime-ID")
	require.NoError(t, err)

	memoryStorage := storage.NewMemoryStorage()
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewCheckKymaResourceDeletedStep(memoryStorage.Operations(), kcpClient)

	// When
	_, backoff, err := step.Run(operation, logger.NewLogSpy().Logger)

	// Then
	assert.Zero(t, backoff)
	assertNoKymaResourceWithGivenRuntimeID(t, kcpClient, operation.KymaResourceNamespace, steps.KymaName(operation))
}

func TestCheckKymaResourceDeleted_RetryWhenStillExists(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = "kyma-system"
	operation.KymaTemplate = kymaTemplate

	kcpClient := fake.NewClientBuilder().Build()

	err := fixKymaResourceWithGivenRuntimeID(kcpClient, operation.KymaResourceNamespace, operation.RuntimeID)
	require.NoError(t, err)

	memoryStorage := storage.NewMemoryStorage()
	err = memoryStorage.Operations().InsertOperation(operation)
	require.NoError(t, err)

	step := NewCheckKymaResourceDeletedStep(memoryStorage.Operations(), kcpClient)

	// When
	_, backoff, err := step.Run(operation, logger.NewLogSpy().Logger)

	// Then
	require.NoError(t, err)
	assert.NotZero(t, backoff)
}

func assertNoKymaResourceWithGivenRuntimeID(t *testing.T, kcpClient client.Client, kymaResourceNamespace string, resourceName string) {
	kymaUnstructured := &unstructured.Unstructured{}
	kymaUnstructured.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operator.kyma-project.io",
		Version: "v1beta2",
		Kind:    "kyma",
	})
	err := kcpClient.Get(context.Background(), client.ObjectKey{
		Namespace: kymaResourceNamespace,
		Name:      resourceName,
	}, kymaUnstructured)
	assert.True(t, errors.IsNotFound(err))
}

func fixKymaResourceWithGivenRuntimeID(kcpClient client.Client, kymaResourceNamespace string, resourceName string) error {
	return kcpClient.Create(context.Background(), &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "operator.kyma-project.io/v1beta2",
		"kind":       "Kyma",
		"metadata": map[string]interface{}{
			"name":      resourceName,
			"namespace": kymaResourceNamespace,
		},
		"spec": map[string]interface{}{
			"channel": "stable",
		},
	}})
}
