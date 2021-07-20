package hyperscaler

import (
	"context"
	"testing"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_fake "github.com/gardener/gardener/pkg/client/core/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGardenerCredentials(t *testing.T) {
	t.Run("should return error if account pool is not configured", func(t *testing.T) {
		//given
		accountProvider := NewAccountProvider(nil, nil, nil)

		//when
		_, err := accountProvider.GardenerSecretName(GCP, "tenantname")
		require.Error(t, err)

		//then
		assert.Contains(t, err.Error(), "Gardener Account pool is not configured")
	})

	t.Run("should return correct credentials", func(t *testing.T) {
		//given
		mockClient := fake.NewSimpleClientset(newSecret("secret1"))
		gardenerFake := gardener_fake.NewSimpleClientset(newSecretBinding("secretBinding1", "secret1", "azure", false))
		mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
		mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)
		accountPool := NewAccountPool(mockSecretBindings, mockShoots)

		accountProvider := NewAccountProvider(mockClient, accountPool, nil)

		//when
		secret, err := accountProvider.GardenerSecretName(Azure, "tenantname")

		//then
		require.NoError(t, err)
		assert.Equal(t, secret, "secret1")
	})

	t.Run("should return correct shared credentials when secret is in another namespace", func(t *testing.T) {
		//given
		mockClient := fake.NewSimpleClientset(&corev1.Secret{
			ObjectMeta: machineryv1.ObjectMeta{
				Name: "secret1", Namespace: "anothernamespace",
			},
			Data: map[string][]byte{
				"credentials": []byte("secret1"),
			},
		})
		gardenerFake := gardener_fake.NewSimpleClientset(&gardener_types.SecretBinding{
			ObjectMeta: machineryv1.ObjectMeta{
				Name: "secretBinding1", Namespace: testNamespace,
				Labels: map[string]string{
					"hyperscalerType": "azure",
				},
			},
			SecretRef: corev1.SecretReference{
				Name:      "secret1",
				Namespace: "anothernamespace",
			},
		})
		mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
		mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)
		accountPool := NewAccountPool(mockSecretBindings, mockShoots)

		accountProvider := NewAccountProvider(mockClient, accountPool, nil)

		//when
		secret, err := accountProvider.GardenerSecretName(Azure, "tenantname")

		//then
		require.NoError(t, err)
		assert.Equal(t, secret, "secret1")
	})

	t.Run("should return error when failed to find secret binding", func(t *testing.T) {
		//given
		mockClient := fake.NewSimpleClientset(newSecret("secret1"))
		gardenerFake := gardener_fake.NewSimpleClientset()
		mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
		mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)
		accountPool := NewAccountPool(mockSecretBindings, mockShoots)

		accountProvider := NewAccountProvider(mockClient, accountPool, nil)

		//when
		_, err := accountProvider.GardenerSecretName(Azure, "tenantname")

		//then
		require.Error(t, err)
	})
}

func TestGardenerSharedCredentials(t *testing.T) {
	t.Run("should return error if shared account pool is not configured", func(t *testing.T) {
		//given
		accountProvider := NewAccountProvider(nil, nil, nil)

		//when
		_, err := accountProvider.GardenerSharedSecretName(GCP)
		require.Error(t, err)

		//then
		assert.Contains(t, err.Error(), "Gardener Shared Account pool is not configured")
	})

	t.Run("should return correct shared credentials", func(t *testing.T) {
		//given
		mockClient := fake.NewSimpleClientset(newSecret("secret1"))
		gardenerFake := gardener_fake.NewSimpleClientset(newSecretBinding("secretBinding1", "secret1", "azure", true))
		mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
		mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)
		sharedAccountPool := NewSharedGardenerAccountPool(mockSecretBindings, mockShoots)

		accountProvider := NewAccountProvider(mockClient, nil, sharedAccountPool)

		//when
		secret, err := accountProvider.GardenerSharedSecretName(Azure)

		//then
		require.NoError(t, err)
		assert.Equal(t, secret, "secret1")
	})

	t.Run("should return correct shared credentials when secret is in another namespace", func(t *testing.T) {
		//given
		mockClient := fake.NewSimpleClientset(&corev1.Secret{
			ObjectMeta: machineryv1.ObjectMeta{
				Name: "secret1", Namespace: "anothernamespace",
			},
			Data: map[string][]byte{
				"credentials": []byte("secret1"),
			},
		})
		gardenerFake := gardener_fake.NewSimpleClientset(&gardener_types.SecretBinding{
			ObjectMeta: machineryv1.ObjectMeta{
				Name: "secretBinding1", Namespace: testNamespace,
				Labels: map[string]string{
					"hyperscalerType": "azure",
					"shared":          "true",
				},
			},
			SecretRef: corev1.SecretReference{
				Name:      "secret1",
				Namespace: "anothernamespace",
			},
		})
		mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
		mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)
		sharedAccountPool := NewSharedGardenerAccountPool(mockSecretBindings, mockShoots)

		accountProvider := NewAccountProvider(mockClient, nil, sharedAccountPool)

		//when
		secret, err := accountProvider.GardenerSharedSecretName(Azure)

		//then
		require.NoError(t, err)
		assert.Equal(t, secret, "secret1")
	})

	t.Run("should return error when failed to find secret binding", func(t *testing.T) {
		//given
		mockClient := fake.NewSimpleClientset(newSecret("secret1"))
		gardenerFake := gardener_fake.NewSimpleClientset()
		mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
		mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)
		sharedAccountPool := NewSharedGardenerAccountPool(mockSecretBindings, mockShoots)

		accountProvider := NewAccountProvider(mockClient, nil, sharedAccountPool)

		//when
		_, err := accountProvider.GardenerSharedSecretName(Azure)

		//then
		require.Error(t, err)
	})
}

func TestMarkUnusedGardenerSecretBindingAsDirty(t *testing.T) {
	t.Run("should mark secret binding as dirty if unused", func(t *testing.T) {
		//given
		pool, secretBindingMock := newTestAccountPoolWithoutShoots()

		accountProvider := NewAccountProvider(nil, pool, nil)

		//when
		err := accountProvider.MarkUnusedGardenerSecretBindingAsDirty(Type("azure"), "tenant1")

		//then
		require.NoError(t, err)
		secretBinding, err := secretBindingMock.Get(context.Background(), "secretBinding1", machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, secretBinding.Labels["dirty"], "true")
	})

	t.Run("should not mark secret binding as dirty if internal", func(t *testing.T) {
		//given
		pool, secretBindingMock := newTestAccountPoolWithSecretBindingInternal()

		accountProvider := NewAccountProvider(nil, pool, nil)

		//when
		err := accountProvider.MarkUnusedGardenerSecretBindingAsDirty(Type("azure"), "tenant1")

		//then
		require.NoError(t, err)
		secretBinding, err := secretBindingMock.Get(context.Background(), "secretBinding1", machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, secretBinding.Labels["dirty"], "")
	})

	t.Run("should not mark secret binding as dirty if used by a cluster", func(t *testing.T) {
		//given
		pool, secretBindingMock := newTestAccountPoolWithSingleShoot()

		accountProvider := NewAccountProvider(nil, pool, nil)

		//when
		err := accountProvider.MarkUnusedGardenerSecretBindingAsDirty(Type("azure"), "tenant1")

		//then
		require.NoError(t, err)
		secretBinding, err := secretBindingMock.Get(context.Background(), "secretBinding1", machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, secretBinding.Labels["dirty"], "")
	})

	t.Run("should not modify a secret binding if marked as dirty", func(t *testing.T) {
		//given
		pool, secretBindingMock := newTestAccountPoolWithSecretBindingDirty()

		accountProvider := NewAccountProvider(nil, pool, nil)

		//when
		err := accountProvider.MarkUnusedGardenerSecretBindingAsDirty(Type("azure"), "tenant1")

		//then
		require.NoError(t, err)
		secretBinding, err := secretBindingMock.Get(context.Background(), "secretBinding1", machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, secretBinding.Labels["dirty"], "true")
	})

	t.Run("should not mark secret binding as dirty if used by multiple cluster", func(t *testing.T) {
		//given
		pool, secretBindingMock := newTestAccountPoolWithShootsUsingSecretBinding()

		accountProvider := NewAccountProvider(nil, pool, nil)

		//when
		err := accountProvider.MarkUnusedGardenerSecretBindingAsDirty(Type("azure"), "tenant1")

		//then
		require.NoError(t, err)
		secretBinding, err := secretBindingMock.Get(context.Background(), "secretBinding1", machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, secretBinding.Labels["dirty"], "")
	})

	t.Run("should return error if failed to read secrets for particular hyperscaler type", func(t *testing.T) {
		//given
		accountProvider := NewAccountProvider(nil, nil, nil)

		//when
		err := accountProvider.MarkUnusedGardenerSecretBindingAsDirty(Type("gcp"), "tenant1")

		//when
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to release subscription for tenant. Gardener Account pool is not configured")
	})
}
