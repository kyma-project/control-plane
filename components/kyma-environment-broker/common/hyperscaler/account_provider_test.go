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
)

func TestGardenerSecretName(t *testing.T) {
	t.Run("should return error if account pool is not configured", func(t *testing.T) {
		//given
		accountProvider := NewAccountProvider(nil, nil)

		//when
		_, err := accountProvider.GardenerSecretName(GCP, "tenantname")
		require.Error(t, err)

		//then
		assert.Contains(t, err.Error(), "Gardener Account pool is not configured")
	})

	t.Run("should return correct secret name", func(t *testing.T) {
		//given
		gardenerFake := gardener_fake.NewSimpleClientset(newSecretBinding("secretBinding1", "secret1", "azure", false))
		mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
		mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)
		accountPool := NewAccountPool(mockSecretBindings, mockShoots)

		accountProvider := NewAccountProvider(accountPool, nil)

		//when
		secretName, err := accountProvider.GardenerSecretName(Azure, "tenantname")

		//then
		require.NoError(t, err)
		assert.Equal(t, secretName, "secret1")
	})

	t.Run("should return correct shared secret name when secret is in another namespace", func(t *testing.T) {
		//given
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

		accountProvider := NewAccountProvider(accountPool, nil)

		//when
		secretName, err := accountProvider.GardenerSecretName(Azure, "tenantname")

		//then
		require.NoError(t, err)
		assert.Equal(t, secretName, "secret1")
	})

	t.Run("should return error when failed to find secret binding", func(t *testing.T) {
		//given
		gardenerFake := gardener_fake.NewSimpleClientset()
		mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
		mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)
		accountPool := NewAccountPool(mockSecretBindings, mockShoots)

		accountProvider := NewAccountProvider(accountPool, nil)

		//when
		_, err := accountProvider.GardenerSecretName(Azure, "tenantname")

		//then
		require.Error(t, err)
	})
}

func TestGardenerSharedSecretName(t *testing.T) {
	t.Run("should return error if shared account pool is not configured", func(t *testing.T) {
		//given
		accountProvider := NewAccountProvider(nil, nil)

		//when
		_, err := accountProvider.GardenerSharedSecretName(GCP)
		require.Error(t, err)

		//then
		assert.Contains(t, err.Error(), "Gardener Shared Account pool is not configured")
	})

	t.Run("should return correct shared secret name", func(t *testing.T) {
		//given
		gardenerFake := gardener_fake.NewSimpleClientset(newSecretBinding("secretBinding1", "secret1", "azure", true))
		mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
		mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)
		sharedAccountPool := NewSharedGardenerAccountPool(mockSecretBindings, mockShoots)

		accountProvider := NewAccountProvider(nil, sharedAccountPool)

		//when
		secretName, err := accountProvider.GardenerSharedSecretName(Azure)

		//then
		require.NoError(t, err)
		assert.Equal(t, secretName, "secret1")
	})

	t.Run("should return correct shared secret name when secret is in another namespace", func(t *testing.T) {
		//given
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

		accountProvider := NewAccountProvider(nil, sharedAccountPool)

		//when
		secretName, err := accountProvider.GardenerSharedSecretName(Azure)

		//then
		require.NoError(t, err)
		assert.Equal(t, secretName, "secret1")
	})

	t.Run("should return error when failed to find secret binding", func(t *testing.T) {
		//given
		gardenerFake := gardener_fake.NewSimpleClientset()
		mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(testNamespace)
		mockShoots := gardenerFake.CoreV1beta1().Shoots(testNamespace)
		sharedAccountPool := NewSharedGardenerAccountPool(mockSecretBindings, mockShoots)

		accountProvider := NewAccountProvider(nil, sharedAccountPool)

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

		accountProvider := NewAccountProvider(pool, nil)

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

		accountProvider := NewAccountProvider(pool, nil)

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

		accountProvider := NewAccountProvider(pool, nil)

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

		accountProvider := NewAccountProvider(pool, nil)

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

		accountProvider := NewAccountProvider(pool, nil)

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
		accountProvider := NewAccountProvider(nil, nil)

		//when
		err := accountProvider.MarkUnusedGardenerSecretBindingAsDirty(Type("gcp"), "tenant1")

		//when
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to release subscription for tenant. Gardener Account pool is not configured")
	})
}
