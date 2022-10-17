package provisioning

import (
	"context"
	"reflect"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apicorev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestInjectBTPOperatorCredentialsStep(t *testing.T) {
	t.Run("should execute step flawlessly", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()

		scheme := internal.NewSchemeForTests()
		err := apiextensionsv1.AddToScheme(scheme)

		k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		operation := fixProvisioningOperationWithClusterIDAndCredentials(k8sClient)
		expectedSecretData := createExpectedSecretData(operation.ProvisioningParameters.ErsContext.SMOperatorCredentials, operation.ServiceManagerClusterID)

		step := NewInjectBTPOperatorCredentialsStep(memoryStorage.Operations(), func(k string) (client.Client, error) { return k8sClient, nil })

		// when
		entry := log.WithFields(logrus.Fields{"step": "TEST"})
		_, _, err = step.Run(operation, entry)

		// then
		assert.NoError(t, err)
		assertTheSecretIsAsExpected(t, k8sClient, expectedSecretData)

		// when
		operation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret = "rotated-sample-client-secret"
		expectedRotatedSecretData := createExpectedSecretData(operation.ProvisioningParameters.ErsContext.SMOperatorCredentials, operation.ServiceManagerClusterID)
		_, _, err = step.Run(operation, entry)

		// then
		assert.NoError(t, err)
		assertTheSecretIsAsExpected(t, k8sClient, expectedRotatedSecretData)
	})
	t.Run("should fail when RuntimeID is empty", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()

		scheme := internal.NewSchemeForTests()
		apiextensionsv1.AddToScheme(scheme)

		k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		operation := fixture.FixProvisioningOperation("operation-id", "inst-id")
		operation.RuntimeID = ""

		step := NewInjectBTPOperatorCredentialsStep(memoryStorage.Operations(), func(k string) (client.Client, error) { return k8sClient, nil })

		// when
		entry := log.WithFields(logrus.Fields{"step": "TEST"})
		processedOperation, _, _ := step.Run(operation, entry)

		// then
		assert.Equal(t, domain.Failed, processedOperation.State)
	})
}

func TestInjectBTPOperatorCredentialsWhenSecretAlreadyExistsStep(t *testing.T) {
	t.Run("should overwrite secret created by user", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()

		userSecret := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"name":      "sap-btp-manager",
				"namespace": "kyma-system",
			},
		}}

		scheme := internal.NewSchemeForTests()
		err := apiextensionsv1.AddToScheme(scheme)

		k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		err = k8sClient.Create(context.TODO(), userSecret)
		require.NoError(t, err)

		operation := fixProvisioningOperationWithClusterIDAndCredentials(k8sClient)
		expectedSecretData := createExpectedSecretData(operation.ProvisioningParameters.ErsContext.SMOperatorCredentials, operation.ServiceManagerClusterID)

		step := NewInjectBTPOperatorCredentialsStep(memoryStorage.Operations(), func(k string) (client.Client, error) { return k8sClient, nil })

		// when
		entry := log.WithFields(logrus.Fields{"step": "TEST"})
		_, _, err = step.Run(operation, entry)

		// then
		assert.NoError(t, err)
		assertTheSecretIsAsExpected(t, k8sClient, expectedSecretData)
	})
}

func fixProvisioningOperationWithClusterIDAndCredentials(k8sClient client.WithWatch) internal.Operation {
	operation := fixProvisioningOperationWithCredentials(k8sClient)
	operation.InstanceDetails.ServiceManagerClusterID = "cluster-id"
	return operation
}

func fixProvisioningOperationWithCredentials(k8sClient client.WithWatch) internal.Operation {
	operation := fixture.FixProvisioningOperation("operation-id", "inst-id")
	operation.K8sClient = k8sClient
	operation.ProvisioningParameters.ErsContext.SMOperatorCredentials = &internal.ServiceManagerOperatorCredentials{
		ClientID:          "sample-client-id",
		ClientSecret:      "sample-client-secret",
		ServiceManagerURL: "www.service.manager.url.com",
		URL:               "www.sample.url.com",
		XSAppName:         "sample-app-name",
	}
	return operation
}

func assertTheSecretIsAsExpected(t *testing.T, k8sClient client.WithWatch, expected map[string]string) {
	secretFromCluster := apicorev1.Secret{}
	err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: secretNamespace, Name: secretName}, &secretFromCluster)
	require.NoError(t, err)
	assert.True(t, reflect.DeepEqual(expected, secretFromCluster.StringData))
	assert.True(t, reflect.DeepEqual(labels, secretFromCluster.Labels))
	assert.True(t, reflect.DeepEqual(annotations, secretFromCluster.Annotations))
}

func createExpectedSecretData(credentials *internal.ServiceManagerOperatorCredentials, clusterID string) map[string]string {
	return map[string]string{
		"clientid":     credentials.ClientID,
		"clientsecret": credentials.ClientSecret,
		"sm_url":       credentials.ServiceManagerURL,
		"tokenurl":     credentials.URL,
		"cluster_id":   clusterID}
}
