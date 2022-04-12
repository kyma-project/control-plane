package job

import (
	"context"
	"testing"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_fake "github.com/gardener/gardener/pkg/client/core/clientset/versioned/fake"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/subscriptioncleanup/cloudprovider/mocks"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/subscriptioncleanup/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var namespace = "test_gardener"

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
		secretBinding := &gardener_types.SecretBinding{
			ObjectMeta: machineryv1.ObjectMeta{
				Name:      "secretBinding1",
				Namespace: namespace,
				Labels: map[string]string{
					"tenantName":      "tenant1",
					"hyperscalerType": "azure",
					"dirty":           "true",
				},
			},
			SecretRef: v1.SecretReference{
				Name:      "secret1",
				Namespace: namespace,
			},
		}

		mockClient := fake.NewSimpleClientset(secret)

		gardenerFake := gardener_fake.NewSimpleClientset(secretBinding)
		mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(namespace)
		mockShoots := gardenerFake.CoreV1beta1().Shoots(namespace)

		resCleaner := &azureMockResourceCleaner{}
		providerFactory := &mocks.ProviderFactory{}
		providerFactory.On("New", model.Azure, mock.Anything).Return(resCleaner, nil)

		cleaner := NewCleaner(context.Background(), mockClient, mockSecretBindings, mockShoots, providerFactory)

		//when
		err := cleaner.Do()

		//then
		require.NoError(t, err)
		cleanedSecretBinding, err := mockSecretBindings.Get(context.Background(), secretBinding.Name, machineryv1.GetOptions{})
		require.NoError(t, err)

		assert.Equal(t, "", cleanedSecretBinding.Labels["dirty"])
		assert.Equal(t, "", cleanedSecretBinding.Labels["tenantName"])
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
		secretBinding := &gardener_types.SecretBinding{
			ObjectMeta: machineryv1.ObjectMeta{
				Name:      "secretBinding1",
				Namespace: namespace,
				Labels: map[string]string{
					"tenantName":      "tenant1",
					"hyperscalerType": "azure",
					"dirty":           "true",
				},
			},
			SecretRef: v1.SecretReference{
				Name:      "secret1",
				Namespace: namespace,
			},
		}

		shoot := &gardener_types.Shoot{
			ObjectMeta: machineryv1.ObjectMeta{
				Name:      "some-name",
				Namespace: namespace,
			},
			Spec: gardener_types.ShootSpec{
				SecretBindingName: secretBinding.Name,
			},
			Status: gardener_types.ShootStatus{},
		}

		mockClient := fake.NewSimpleClientset(secret)

		gardenerFake := gardener_fake.NewSimpleClientset(secretBinding, shoot)
		mockSecretBindings := gardenerFake.CoreV1beta1().SecretBindings(namespace)
		mockShoots := gardenerFake.CoreV1beta1().Shoots(namespace)

		resCleaner := &azureMockResourceCleaner{}
		providerFactory := &mocks.ProviderFactory{}
		providerFactory.On("New", model.Azure, mock.Anything).Return(resCleaner, nil)

		cleaner := NewCleaner(context.Background(), mockClient, mockSecretBindings, mockShoots, providerFactory)

		//when
		err := cleaner.Do()

		//then
		require.NoError(t, err)
		cleanedSecretBinding, err := mockSecretBindings.Get(context.Background(), secretBinding.Name, machineryv1.GetOptions{})
		require.NoError(t, err)

		assert.Equal(t, "true", cleanedSecretBinding.Labels["dirty"])
		assert.Equal(t, "tenant1", cleanedSecretBinding.Labels["tenantName"])
	})
}

type azureMockResourceCleaner struct {
	error error
}

func (am *azureMockResourceCleaner) Do() error {
	return am.error
}
