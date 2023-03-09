package runtime

import (
	"context"

	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	"github.com/kyma-incubator/compass/components/director/pkg/graphql"
	mocks2 "github.com/kyma-project/control-plane/components/provisioner/internal/director/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	kubeconfig = "some Kubeconfig"
)

func TestProvider_CreateConfigMapForRuntime(t *testing.T) {
	connectorURL := "https://kyma.cx/connector/graphql"
	runtimeID := "123-123-456"
	tenant := "tenant"
	token := "shdfv7123ygfbw832b"

	namespace := "kyma-system"
	legacyNamespace := "compass-system"

	cluster := model.Cluster{
		ID:     runtimeID,
		Tenant: tenant,
	}

	oneTimeToken := graphql.OneTimeTokenForRuntimeExt{
		OneTimeTokenForRuntime: graphql.OneTimeTokenForRuntime{
			TokenWithURL: graphql.TokenWithURL{Token: token, ConnectorURL: connectorURL},
		},
	}

	t.Run("Should configure Runtime Agent", func(t *testing.T) {
		//given
		k8sClientProvider := newMockClientProvider(t)
		directorClient := &mocks2.DirectorClient{}

		directorClient.On("GetConnectionToken", runtimeID, tenant).Return(oneTimeToken, nil)

		configProvider := NewRuntimeConfigurator(k8sClientProvider, directorClient)

		//when
		err := configProvider.ConfigureRuntime(cluster, kubeconfig)

		//then
		require.NoError(t, err)
		secret, k8serr := k8sClientProvider.fakeClient.CoreV1().Secrets(namespace).Get(context.Background(), AgentConfigurationSecretName, v1.GetOptions{})
		secretLegacyNamespace, k8serr := k8sClientProvider.fakeClient.CoreV1().Secrets(legacyNamespace).Get(context.Background(), AgentConfigurationSecretName, v1.GetOptions{})
		require.NoError(t, k8serr)

		assertData := func(data map[string]string) {
			assert.Equal(t, connectorURL, data["CONNECTOR_URL"])
			assert.Equal(t, runtimeID, data["RUNTIME_ID"])
			assert.Equal(t, tenant, data["TENANT"])
			assert.Equal(t, token, data["TOKEN"])
		}

		assertData(secret.StringData)
		assertData(secretLegacyNamespace.StringData)
	})
	t.Run("Should reconfigure Runtime Agent", func(t *testing.T) {
		//given
		k8sClientProvider := newMockClientProvider(t)
		oldSecret := &core.Secret{
			ObjectMeta: meta.ObjectMeta{
				Name:      AgentConfigurationSecretName,
				Namespace: namespace,
			},
			StringData: map[string]string{
				"key": "value",
			},
		}
		legacyNamespaceOldSecret := &core.Secret{
			ObjectMeta: meta.ObjectMeta{
				Name:      AgentConfigurationSecretName,
				Namespace: legacyNamespace,
			},
			StringData: map[string]string{
				"key": "value",
			},
		}
		secret, k8serr := k8sClientProvider.fakeClient.CoreV1().Secrets(namespace).Create(context.Background(), oldSecret, v1.CreateOptions{})
		secretLegacyNamespace, k8serr := k8sClientProvider.fakeClient.CoreV1().Secrets(legacyNamespace).Create(context.Background(), legacyNamespaceOldSecret, v1.CreateOptions{})
		require.NoError(t, k8serr)

		directorClient := &mocks2.DirectorClient{}
		directorClient.On("GetConnectionToken", runtimeID, tenant).Return(oneTimeToken, nil)

		configProvider := NewRuntimeConfigurator(k8sClientProvider, directorClient)

		//when
		err := configProvider.ConfigureRuntime(cluster, kubeconfig)

		//then
		require.NoError(t, err)
		secret, k8serr = k8sClientProvider.fakeClient.CoreV1().Secrets(namespace).Get(context.Background(), AgentConfigurationSecretName, v1.GetOptions{})
		secretLegacyNamespace, k8serr = k8sClientProvider.fakeClient.CoreV1().Secrets(legacyNamespace).Get(context.Background(), AgentConfigurationSecretName, v1.GetOptions{})
		require.NoError(t, k8serr)

		assertData := func(data map[string]string) {
			assert.Equal(t, connectorURL, data["CONNECTOR_URL"])
			assert.Equal(t, runtimeID, data["RUNTIME_ID"])
			assert.Equal(t, tenant, data["TENANT"])
			assert.Equal(t, token, data["TOKEN"])
		}

		assertData(secret.StringData)
		assertData(secretLegacyNamespace.StringData)
	})

	t.Run("Should retry on GetConnectionToken and configure Runtime Agent", func(t *testing.T) {
		//given
		k8sClientProvider := newMockClientProvider(t)
		directorClient := &mocks2.DirectorClient{}

		directorClient.On("GetConnectionToken", runtimeID, tenant).Once().Return(graphql.OneTimeTokenForRuntimeExt{}, apperrors.Internal("token error"))
		directorClient.On("GetConnectionToken", runtimeID, tenant).Once().Return(oneTimeToken, nil)

		configProvider := NewRuntimeConfigurator(k8sClientProvider, directorClient)

		//when
		err := configProvider.ConfigureRuntime(cluster, kubeconfig)

		//then
		require.NoError(t, err)
		secret, k8serr := k8sClientProvider.fakeClient.CoreV1().Secrets(namespace).Get(context.Background(), AgentConfigurationSecretName, v1.GetOptions{})
		secretLegacyNamespace, k8serr := k8sClientProvider.fakeClient.CoreV1().Secrets(legacyNamespace).Get(context.Background(), AgentConfigurationSecretName, v1.GetOptions{})
		require.NoError(t, k8serr)

		assertData := func(data map[string]string) {
			assert.Equal(t, connectorURL, data["CONNECTOR_URL"])
			assert.Equal(t, runtimeID, data["RUNTIME_ID"])
			assert.Equal(t, tenant, data["TENANT"])
			assert.Equal(t, token, data["TOKEN"])
		}

		assertData(secret.StringData)
		assertData(secretLegacyNamespace.StringData)
	})

	t.Run("Should return error when failed to create client", func(t *testing.T) {
		//given

		k8sClientProvider := newErrorClientProvider(t, apperrors.Internal("error"))
		directorClient := &mocks2.DirectorClient{}

		directorClient.On("GetConnectionToken", runtimeID, tenant).Return(oneTimeToken, nil)

		configProvider := NewRuntimeConfigurator(k8sClientProvider, directorClient)

		//when
		err := configProvider.ConfigureRuntime(cluster, kubeconfig)

		//then
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeInternal)
	})

	t.Run("Should return error when failed to fetch token", func(t *testing.T) {
		//given
		directorClient := &mocks2.DirectorClient{}

		directorClient.On("GetConnectionToken", runtimeID, tenant).Return(graphql.OneTimeTokenForRuntimeExt{}, apperrors.Internal("error"))

		configProvider := NewRuntimeConfigurator(nil, directorClient)

		//when
		err := configProvider.ConfigureRuntime(cluster, kubeconfig)

		//then
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeInternal)
	})
}

type mockClientProvider struct {
	t          *testing.T
	fakeClient *fake.Clientset
	err        apperrors.AppError
}

func newMockClientProvider(t *testing.T, objects ...runtime.Object) *mockClientProvider {
	return &mockClientProvider{
		t:          t,
		fakeClient: fake.NewSimpleClientset(objects...),
	}
}

func newErrorClientProvider(t *testing.T, err apperrors.AppError) *mockClientProvider {
	return &mockClientProvider{
		t:   t,
		err: err,
	}
}

func (m *mockClientProvider) CreateK8SClient(kubeconfigRaw string) (kubernetes.Interface, apperrors.AppError) {
	assert.Equal(m.t, kubeconfig, kubeconfigRaw)

	if m.err != nil {
		return nil, m.err
	}

	return m.fakeClient, nil
}
