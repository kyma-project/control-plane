package hyperscaler

import (
	"context"
	"fmt"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestGardenerSecretName(t *testing.T) {
	t.Run("should return error if account pool is not configured", func(t *testing.T) {
		//given
		accountProvider := NewAccountProvider(nil, nil)

		//when
		_, err := accountProvider.GardenerSecretName(GCP, "tenantname", false)
		require.Error(t, err)

		//then
		assert.Contains(t, err.Error(), "Gardener Account pool is not configured")
	})

	t.Run("should return correct secret name", func(t *testing.T) {
		//given
		gardenerFake := gardener.NewDynamicFakeClient(newSecretBinding("secretBinding1", "secret1", "azure", false, false))
		accountPool := NewAccountPool(gardenerFake, testNamespace)

		accountProvider := NewAccountProvider(accountPool, nil)

		//when
		secretName, err := accountProvider.GardenerSecretName(Azure, "tenantname", false)

		//then
		require.NoError(t, err)
		assert.Equal(t, secretName, "secret1")
	})

	t.Run("should return correct shared secret name when secret is in another namespace", func(t *testing.T) {
		//given
		sb := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "secretBinding1",
					"namespace": testNamespace,
					"labels": map[string]interface{}{
						"hyperscalerType": "azure",
					},
				},
				"secretRef": map[string]interface{}{
					"name":      "secret1",
					"namespace": "anothernamespace",
				},
			},
		}
		sb.SetGroupVersionKind(secretBindingGVK)
		gardenerFake := gardener.NewDynamicFakeClient(sb)
		accountPool := NewAccountPool(gardenerFake, testNamespace)

		accountProvider := NewAccountProvider(accountPool, nil)

		//when
		secretName, err := accountProvider.GardenerSecretName(Azure, "tenantname", false)

		//then
		require.NoError(t, err)
		assert.Equal(t, secretName, "secret1")
	})

	t.Run("should return error when failed to find secret binding", func(t *testing.T) {
		//given
		gardenerFake := gardener.NewDynamicFakeClient()
		accountPool := NewAccountPool(gardenerFake, testNamespace)

		accountProvider := NewAccountProvider(accountPool, nil)

		//when
		_, err := accountProvider.GardenerSecretName(Azure, "tenantname", false)

		//then
		require.Error(t, err)
	})
}

func TestGardenerSharedSecretName(t *testing.T) {
	t.Run("should return error if shared account pool is not configured", func(t *testing.T) {
		//given
		accountProvider := NewAccountProvider(nil, nil)

		//when
		_, err := accountProvider.GardenerSharedSecretName(GCP, false)
		require.Error(t, err)

		//then
		assert.Contains(t, err.Error(), "Gardener Shared Account pool is not configured")
	})

	t.Run("should return correct shared secret name", func(t *testing.T) {
		//given
		gardenerFake := gardener.NewDynamicFakeClient(newSecretBinding("secretBinding1", "secret1", "azure", true, false))
		sharedAccountPool := NewSharedGardenerAccountPool(gardenerFake, testNamespace)

		accountProvider := NewAccountProvider(nil, sharedAccountPool)

		//when
		secretName, err := accountProvider.GardenerSharedSecretName(Azure, false)

		//then
		require.NoError(t, err)
		assert.Equal(t, secretName, "secret1")
	})

	t.Run("should return correct shared secret name when secret is in another namespace", func(t *testing.T) {
		//given
		sb := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "secretBinding1",
					"namespace": testNamespace,
					"labels": map[string]interface{}{
						"hyperscalerType": "azure",
						"shared":          "true",
					},
				},
				"secretRef": map[string]interface{}{
					"name":      "secret1",
					"namespace": "anothernamespace",
				},
			},
		}
		sb.SetGroupVersionKind(secretBindingGVK)
		gardenerFake := gardener.NewDynamicFakeClient(sb)
		sharedAccountPool := NewSharedGardenerAccountPool(gardenerFake, testNamespace)

		accountProvider := NewAccountProvider(nil, sharedAccountPool)

		//when
		secretName, err := accountProvider.GardenerSharedSecretName(Azure, false)

		//then
		require.NoError(t, err)
		assert.Equal(t, secretName, "secret1")
	})

	t.Run("should return error when failed to find secret binding", func(t *testing.T) {
		//given
		gardenerFake := gardener.NewDynamicFakeClient()
		sharedAccountPool := NewSharedGardenerAccountPool(gardenerFake, testNamespace)

		accountProvider := NewAccountProvider(nil, sharedAccountPool)

		//when
		_, err := accountProvider.GardenerSharedSecretName(Azure, false)

		//then
		require.Error(t, err)
	})
}

func TestMarkUnusedGardenerSecretBindingAsDirty(t *testing.T) {

	for _, euAccess := range []bool{false, true} {
		t.Run(fmt.Sprintf("EuAccess=%v", euAccess), func(t *testing.T) {
			t.Run("should mark secret binding as dirty if unused", func(t *testing.T) {
				//given
				pool, secretBindingMock := newTestAccountPoolWithoutShoots(euAccess)

				accountProvider := NewAccountProvider(pool, nil)

				//when
				err := accountProvider.MarkUnusedGardenerSecretBindingAsDirty(Type("azure"), "tenant1", euAccess)

				//then
				require.NoError(t, err)
				secretBinding, err := secretBindingMock.Get(context.Background(), "secretBinding1", machineryv1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, secretBinding.GetLabels()["dirty"], "true")
			})

			t.Run("should not mark secret binding as dirty if internal", func(t *testing.T) {
				//given
				pool, secretBindingMock := newTestAccountPoolWithSecretBindingInternal(euAccess)

				accountProvider := NewAccountProvider(pool, nil)

				//when
				err := accountProvider.MarkUnusedGardenerSecretBindingAsDirty(Type("azure"), "tenant1", euAccess)

				//then
				require.NoError(t, err)
				secretBinding, err := secretBindingMock.Get(context.Background(), "secretBinding1", machineryv1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, secretBinding.GetLabels()["dirty"], "")
			})

			t.Run("should not mark secret binding as dirty if used by a cluster", func(t *testing.T) {
				//given
				pool, secretBindingMock := newTestAccountPoolWithSingleShoot(euAccess)

				accountProvider := NewAccountProvider(pool, nil)

				//when
				err := accountProvider.MarkUnusedGardenerSecretBindingAsDirty(Type("azure"), "tenant1", euAccess)

				//then
				require.NoError(t, err)
				secretBinding, err := secretBindingMock.Get(context.Background(), "secretBinding1", machineryv1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, secretBinding.GetLabels()["dirty"], "")
			})

			t.Run("should not modify a secret binding if marked as dirty", func(t *testing.T) {
				//given
				pool, secretBindingMock := newTestAccountPoolWithSecretBindingDirty(euAccess)

				accountProvider := NewAccountProvider(pool, nil)

				//when
				err := accountProvider.MarkUnusedGardenerSecretBindingAsDirty(Type("azure"), "tenant1", euAccess)

				//then
				require.NoError(t, err)
				secretBinding, err := secretBindingMock.Get(context.Background(), "secretBinding1", machineryv1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, secretBinding.GetLabels()["dirty"], "true")
			})

			t.Run("should not mark secret binding as dirty if used by multiple cluster", func(t *testing.T) {
				//given
				pool, secretBindingMock := newTestAccountPoolWithShootsUsingSecretBinding(euAccess)

				accountProvider := NewAccountProvider(pool, nil)

				//when
				err := accountProvider.MarkUnusedGardenerSecretBindingAsDirty(Type("azure"), "tenant1", euAccess)

				//then
				require.NoError(t, err)
				secretBinding, err := secretBindingMock.Get(context.Background(), "secretBinding1", machineryv1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, secretBinding.GetLabels()["dirty"], "")
			})

			t.Run("should return error if failed to read secrets for particular hyperscaler type", func(t *testing.T) {
				//given
				accountProvider := NewAccountProvider(nil, nil)

				//when
				err := accountProvider.MarkUnusedGardenerSecretBindingAsDirty(Type("gcp"), "tenant1", euAccess)

				//when
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to release subscription for tenant tenant1. Gardener Account pool is not configured")
			})
		})
	}
}
