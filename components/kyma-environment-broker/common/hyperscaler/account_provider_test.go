package hyperscaler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGardenerSharedCredentials_Error(t *testing.T) {

	accountProvider := NewAccountProvider(nil, nil)

	_, err := accountProvider.GardenerSharedCredentials(Type("gcp"))
	require.Error(t, err)

	assert.Contains(t, err.Error(), "Gardener Shared Account pool is not configured")
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
		secretBinding, err := secretBindingMock.Get("secretBinding1", machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, secretBinding.Labels["dirty"], "true")
	})

	t.Run("should not mark secret binding as dirty if internal", func(t *testing.T) {
		//given
		pool, secretBindingMock := newTestAccountPoolWithSecretInternal()

		accountProvider := NewAccountProvider(pool, nil)

		//when
		err := accountProvider.MarkUnusedGardenerSecretBindingAsDirty(Type("azure"), "tenant1")

		//then
		require.NoError(t, err)
		secretBinding, err := secretBindingMock.Get("secretBinding1", machineryv1.GetOptions{})
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
		secretBinding, err := secretBindingMock.Get("secretBinding1", machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, secretBinding.Labels["dirty"], "")
	})

	t.Run("should not modify a secret binding if marked as dirty", func(t *testing.T) {
		//given
		pool, secretBindingMock := newTestAccountPoolWithSecretDirty()

		accountProvider := NewAccountProvider(pool, nil)

		//when
		err := accountProvider.MarkUnusedGardenerSecretBindingAsDirty(Type("azure"), "tenant1")

		//then
		require.NoError(t, err)
		secretBinding, err := secretBindingMock.Get("secretBinding1", machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, secretBinding.Labels["dirty"], "true")
	})

	t.Run("should not mark secret binding as dirty if used by multiple cluster", func(t *testing.T) {
		//given
		pool, secretBindingMock := newTestAccountPoolWithShootsUsingSecret()

		accountProvider := NewAccountProvider(pool, nil)

		//when
		err := accountProvider.MarkUnusedGardenerSecretBindingAsDirty(Type("azure"), "tenant1")

		//then
		require.NoError(t, err)
		secretBinding, err := secretBindingMock.Get("secretBinding1", machineryv1.GetOptions{})
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
