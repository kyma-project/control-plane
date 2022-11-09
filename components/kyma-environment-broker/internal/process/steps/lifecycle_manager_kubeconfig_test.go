package steps

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCheckKymaKubeconfigCreated(t *testing.T) {
	// Given
	operation := fixture.FixProvisioningOperation("op", "instance")
	operation.KymaResourceNamespace = "kyma-system"

	k8sClient := fake.NewClientBuilder().Build()

	memoryStorage := storage.NewMemoryStorage()
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := SyncKubeconfig(memoryStorage.Operations(), k8sClient)

	// When
	_, backoff, err := step.Run(operation, logger.NewLogSpy().Logger)

	// Then
	assert.Zero(t, backoff)

	sec := v1.Secret{}
	err = k8sClient.Get(context.Background(), types.NamespacedName{Namespace: "kyma-system", Name: "kubeconfig-runtime-instance"}, &sec)
	assert.NoError(t, err)
}

func TestCheckKymaKubeconfigDeleted(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation("op", "instance")
	operation.KymaResourceNamespace = "kyma-system"

	k8sClient := fake.NewClientBuilder().Build()
	err := k8sClient.Create(context.Background(), &v1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "kyma-system", Name: "kubeconfig-runtime-instance"}})
	assert.NoError(t, err)

	memoryStorage := storage.NewMemoryStorage()
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := DeleteKubeconfig(memoryStorage.Operations(), k8sClient)

	// When
	_, backoff, err := step.Run(operation, logger.NewLogSpy().Logger)

	// Then
	assert.Zero(t, backoff)

	sec := v1.Secret{}
	err = k8sClient.Get(context.Background(), types.NamespacedName{Namespace: "kyma-system", Name: "kubeconfig-runtime-instance"}, &sec)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
}
