package hyperscaler

import (
	"testing"

	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGardenerSharedCredentials_Error(t *testing.T) {

	accountProvider := NewAccountProvider(nil, nil)

	_, err := accountProvider.GardenerSharedCredentials(Type("gcp"))
	require.Error(t, err)

	assert.Contains(t, err.Error(), "Gardener Shared Account pool is not configured")
}

func TestMarkUnusedGardenerSecretAsDirty(t *testing.T) {
	t.Run("should mark secret as dirty if unused", func(t *testing.T) {
		//given
		pool, secretsMock := newTestAccountPoolWithoutShoots()

		accountProvider := NewAccountProvider(pool, nil)

		//when
		err := accountProvider.MarkUnusedGardenerSecretAsDirty(Type("azure"), "tenant1")

		//then
		require.NoError(t, err)
		secret, err := secretsMock.Get("secret1", machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, secret.Labels["dirty"], "true")
	})

	t.Run("should not mark secret as dirty if used by a cluster", func(t *testing.T) {
		//given
		pool, secretMock := newTestAccountPoolWithSingleShoot()

		accountProvider := NewAccountProvider(pool, nil)

		//when
		err := accountProvider.MarkUnusedGardenerSecretAsDirty(Type("azure"), "tenant1")

		//then
		require.NoError(t, err)
		secret, err := secretMock.Get("secret1", machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, secret.Labels["dirty"], "")
	})

	t.Run("should not modify a secret if marked as dirty", func(t *testing.T) {
		//given
		pool, secretsMock := newTestAccountPoolWithSecretDirty()

		accountProvider := NewAccountProvider(pool, nil)

		//when
		err := accountProvider.MarkUnusedGardenerSecretAsDirty(Type("azure"), "tenant1")

		//then
		require.NoError(t, err)
		secret, err := secretsMock.Get("secret1", machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, secret.Labels["dirty"], "true")
	})

	t.Run("should not mark secret as dirty if used by multiple cluster", func(t *testing.T) {
		//given
		pool, secretsMock := newTestAccountPoolWithShootsUsingSecret()

		accountProvider := NewAccountProvider(pool, nil)

		//when
		err := accountProvider.MarkUnusedGardenerSecretAsDirty(Type("azure"), "tenant1")

		//then
		require.NoError(t, err)
		secret, err := secretsMock.Get("secret1", machineryv1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, secret.Labels["dirty"], "")
	})

	t.Run("should return error if failed to read secrets for particular hyperscaler type", func(t *testing.T) {
		//given
		accountProvider := NewAccountProvider(nil, nil)

		//when
		err := accountProvider.MarkUnusedGardenerSecretAsDirty(Type("gcp"), "tenant1")

		//when
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to release subscription for tenant. Gardener Account pool is not configured")
	})
}
