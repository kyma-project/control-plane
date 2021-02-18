package job

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/subscription-cleanup-job/internal/cloudprovider/mocks"
	"github.com/kyma-project/control-plane/components/subscription-cleanup-job/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var namespace = "test_gardener"

func TestCleanerJob(t *testing.T) {
	t.Run("should return secret to the secrets pool", func(t *testing.T) {
		//given
		secret := &v1.Secret{
			ObjectMeta: machineryv1.ObjectMeta{
				Name: "secret1", Namespace: namespace,
				Labels: map[string]string{
					"tenantName":      "tenant1",
					"hyperscalerType": "azure",
					"dirty":           "true",
				},
			},
			Data: map[string][]byte{
				"credentials":    []byte("secret1"),
				"clientID":       []byte("tenant1"),
				"clientSecret":   []byte("secret"),
				"subscriptionID": []byte("12344"),
				"tenantID":       []byte("tenant1"),
			},
		}

		mockClient := fake.NewSimpleClientset(secret)
		mockSecrets := mockClient.CoreV1().Secrets(namespace)

		resCleaner := &azureMockResourceCleaner{}

		providerFactory := &mocks.ProviderFactory{}
		providerFactory.On("New", model.Azure, mock.Anything).Return(resCleaner, nil)

		cleaner := NewCleaner(context.Background(), mockSecrets, providerFactory)

		//when
		err := cleaner.Do()

		//then
		require.NoError(t, err)
		cleanedSecret, err := mockSecrets.Get(context.Background(), secret.Name, machineryv1.GetOptions{})
		require.NoError(t, err)

		assert.Equal(t, "", cleanedSecret.Labels["dirty"])
		assert.Equal(t, "", cleanedSecret.Labels["tenantName"])
	})
}

type azureMockResourceCleaner struct {
	error error
}

func (am *azureMockResourceCleaner) Do() error {
	return am.error
}
