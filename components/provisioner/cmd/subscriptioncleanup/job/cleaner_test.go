package job

import (
	"context"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/cmd/subscriptioncleanup/cloudprovider/mocks"
	"github.com/kyma-project/kyma-environment-broker/cmd/subscriptioncleanup/model"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	namespace        = "test_gardener"
	shootGVK         = schema.GroupVersionKind{Group: "core.gardener.cloud", Version: "v1beta1", Kind: "Shoot"}
	secretBindingGVK = schema.GroupVersionKind{Group: "core.gardener.cloud", Version: "v1beta1", Kind: "SecretBinding"}
)

func TestCleanerJob(t *testing.T) {
	t.Run("should return secret binding to the secrets pool", func(t *testing.T) {
		//given
		secret := &v1.Secret{
			ObjectMeta: machineryv1.ObjectMeta{
				Name: "secret1", Namespace: namespace,
			},
			Data: map[string][]byte{
				"credentials":    []byte("secret1"),
				"clientID":       []byte("tenant1"),
				"clientSecret":   []byte("secret"),
				"subscriptionID": []byte("12344"),
				"tenantID":       []byte("tenant1"),
			},
		}
		secretBinding := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "secretBinding1",
					"namespace": namespace,
					"labels": map[string]interface{}{
						"tenantName":      "tenant1",
						"hyperscalerType": "azure",
						"dirty":           "true",
					},
				},
				"secretRef": map[string]interface{}{
					"name":      "secret1",
					"namespace": namespace,
				},
			},
		}
		secretBinding.SetGroupVersionKind(secretBindingGVK)

		mockClient := fake.NewSimpleClientset(secret)

		gardenerFake := gardener.NewDynamicFakeClient(secretBinding)
		mockSecretBindings := gardenerFake.Resource(gardener.SecretBindingResource).Namespace(namespace)
		mockShoots := gardenerFake.Resource(gardener.ShootResource).Namespace(namespace)

		resCleaner := &azureMockResourceCleaner{}
		providerFactory := &mocks.ProviderFactory{}
		providerFactory.On("New", model.Azure, mock.Anything).Return(resCleaner, nil)

		cleaner := NewCleaner(context.Background(), mockClient, mockSecretBindings, mockShoots, providerFactory)

		//when
		err := cleaner.Do()

		//then
		require.NoError(t, err)
		cleanedSecretBinding, err := mockSecretBindings.Get(context.Background(), secretBinding.GetName(), machineryv1.GetOptions{})
		require.NoError(t, err)

		assert.Equal(t, "", cleanedSecretBinding.GetLabels()["dirty"])
		assert.Equal(t, "", cleanedSecretBinding.GetLabels()["tenantName"])
	})

	t.Run("should not return secret binding to the secrets pool when secret is still in use", func(t *testing.T) {
		//given
		secret := &v1.Secret{
			ObjectMeta: machineryv1.ObjectMeta{
				Name: "secret1", Namespace: namespace,
			},
			Data: map[string][]byte{
				"credentials":    []byte("secret1"),
				"clientID":       []byte("tenant1"),
				"clientSecret":   []byte("secret"),
				"subscriptionID": []byte("12344"),
				"tenantID":       []byte("tenant1"),
			},
		}
		secretBinding := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "secretBinding1",
					"namespace": namespace,
					"labels": map[string]interface{}{
						"tenantName":      "tenant1",
						"hyperscalerType": "azure",
						"dirty":           "true",
					},
				},
				"secretRef": map[string]interface{}{
					"name":      "secret1",
					"namespace": namespace,
				},
			},
		}
		secretBinding.SetGroupVersionKind(secretBindingGVK)

		shoot := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "some-name",
					"namespace": namespace,
				},
				"spec": map[string]interface{}{
					"secretBindingName": secretBinding.GetName(),
				},
				"status": map[string]interface{}{},
			},
		}
		shoot.SetGroupVersionKind(shootGVK)

		mockClient := fake.NewSimpleClientset(secret)

		gardenerFake := gardener.NewDynamicFakeClient(secretBinding, shoot)
		mockSecretBindings := gardenerFake.Resource(gardener.SecretBindingResource).Namespace(namespace)
		mockShoots := gardenerFake.Resource(gardener.ShootResource).Namespace(namespace)

		resCleaner := &azureMockResourceCleaner{}
		providerFactory := &mocks.ProviderFactory{}
		providerFactory.On("New", model.Azure, mock.Anything).Return(resCleaner, nil)

		cleaner := NewCleaner(context.Background(), mockClient, mockSecretBindings, mockShoots, providerFactory)

		//when
		err := cleaner.Do()

		//then
		require.NoError(t, err)
		cleanedSecretBinding, err := mockSecretBindings.Get(context.Background(), secretBinding.GetName(), machineryv1.GetOptions{})
		require.NoError(t, err)

		assert.Equal(t, "true", cleanedSecretBinding.GetLabels()["dirty"])
		assert.Equal(t, "tenant1", cleanedSecretBinding.GetLabels()["tenantName"])
	})
}

type azureMockResourceCleaner struct {
	error error
}

func (am *azureMockResourceCleaner) Do() error {
	return am.error
}
