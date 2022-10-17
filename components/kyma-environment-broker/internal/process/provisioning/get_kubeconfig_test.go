package provisioning

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	kubeconfigContentsFromParameters = "apiVersion: v1"
	kubeconfigFromRuntime            = "kubeconfig-content"
	kubeconfigFromPreviousOperation  = "kubeconfig-already-set"
)

func TestGetKubeconfigStep(t *testing.T) {
	t.Run("should create k8s client using kubeconfig from RuntimeStatus", func(t *testing.T) {
		// given
		st := storage.NewMemoryStorage()
		provisionerClient := provisioner.NewFakeClient()

		scheme := internal.NewSchemeForTests()
		err := apiextensionsv1.AddToScheme(scheme)

		k8sCli := fake.NewClientBuilder().WithScheme(scheme).Build()

		expectedKubeconfig := kubeconfigFromRuntime
		assertedK8sClientProvider := func(kubeconfig string) (client.Client, error) {
			assert.Equal(t, expectedKubeconfig, kubeconfig)
			return k8sCli, nil
		}
		step := NewGetKubeconfigStep(st.Operations(), provisionerClient, assertedK8sClientProvider)
		operation := fixture.FixProvisioningOperation("operation-id", "inst-id")
		operation.Kubeconfig = ""
		st.Operations().InsertOperation(operation)

		// when
		processedOperation, d, err := step.Run(operation, logrus.New())

		// then
		require.NoError(t, err)
		assert.Zero(t, d)
		assert.Equal(t, kubeconfigFromRuntime, processedOperation.Kubeconfig)
		assert.NotEmpty(t, processedOperation.Kubeconfig)
		assert.NotEmpty(t, processedOperation.K8sClient)
	})
	t.Run("should create k8s client for own_cluster plan using kubeconfig from provisioning parameters", func(t *testing.T) {
		// given
		st := storage.NewMemoryStorage()

		scheme := internal.NewSchemeForTests()
		err := apiextensionsv1.AddToScheme(scheme)

		k8sCli := fake.NewClientBuilder().WithScheme(scheme).Build()

		expectedKubeconfig := kubeconfigContentsFromParameters
		assertedK8sClientProvider := func(kubeconfig string) (client.Client, error) {
			assert.Equal(t, expectedKubeconfig, kubeconfig)
			return k8sCli, nil
		}
		step := NewGetKubeconfigStep(st.Operations(), nil, assertedK8sClientProvider)
		operation := fixture.FixProvisioningOperation("operation-id", "inst-id")
		operation.Kubeconfig = ""
		operation.ProvisioningParameters.Parameters.Kubeconfig = kubeconfigContentsFromParameters
		operation.ProvisioningParameters.PlanID = broker.OwnClusterPlanID
		st.Operations().InsertOperation(operation)

		// when
		processedOperation, d, err := step.Run(operation, logrus.New())

		// then
		require.NoError(t, err)
		assert.Zero(t, d)
		assert.Equal(t, kubeconfigContentsFromParameters, processedOperation.Kubeconfig)
		assert.NotEmpty(t, processedOperation.K8sClient)
	})
	t.Run("should create k8s client using kubeconfig already set in operation", func(t *testing.T) {
		// given
		st := storage.NewMemoryStorage()

		scheme := internal.NewSchemeForTests()
		err := apiextensionsv1.AddToScheme(scheme)

		k8sCli := fake.NewClientBuilder().WithScheme(scheme).Build()

		expectedKubeconfig := kubeconfigFromPreviousOperation

		assertedK8sClientProvider := func(kubeconfig string) (client.Client, error) {
			assert.Equal(t, expectedKubeconfig, kubeconfig)
			return k8sCli, nil
		}
		step := NewGetKubeconfigStep(st.Operations(), nil, assertedK8sClientProvider)
		operation := fixture.FixProvisioningOperation("operation-id", "inst-id")
		operation.Kubeconfig = kubeconfigFromPreviousOperation
		operation.ProvisioningParameters.Parameters.Kubeconfig = ""
		st.Operations().InsertOperation(operation)

		// when
		processedOperation, d, err := step.Run(operation, logrus.New())

		// then
		require.NoError(t, err)
		assert.Zero(t, d)
		assert.Equal(t, kubeconfigFromPreviousOperation, processedOperation.Kubeconfig)
		assert.NotEmpty(t, processedOperation.K8sClient)
	})
	t.Run("should fail with error if there is neither kubeconfig nor runtimeID and this is not own_cluster plan", func(t *testing.T) {
		// given
		st := storage.NewMemoryStorage()
		provisionerClient := provisioner.NewFakeClient()

		scheme := internal.NewSchemeForTests()
		err := apiextensionsv1.AddToScheme(scheme)

		k8sCli := fake.NewClientBuilder().WithScheme(scheme).Build()

		assertedK8sClientProvider := func(kubeconfig string) (client.Client, error) {
			assert.Fail(t, "should not call this assertion")
			return k8sCli, nil
		}
		step := NewGetKubeconfigStep(st.Operations(), provisionerClient, assertedK8sClientProvider)
		operation := fixture.FixProvisioningOperation("operation-id", "inst-id")
		operation.Kubeconfig = ""
		operation.RuntimeID = ""
		st.Operations().InsertOperation(operation)

		// when
		_, _, err = step.Run(operation, logrus.New())

		// then
		require.ErrorContains(t, err, "Runtime ID is empty")
	})
}
