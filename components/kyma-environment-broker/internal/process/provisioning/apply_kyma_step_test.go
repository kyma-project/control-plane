package provisioning

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreatingKymaResource(t *testing.T) {
	// given
	operation := fixOperationForApplyKymaResource(t)
	storage := storage.NewMemoryStorage()
	storage.Operations().InsertOperation(operation)
	svc := NewApplyKymaStep(storage.Operations())

	// when
	_, backoff, err := svc.Run(operation, logger.NewLogSpy().Logger)

	// then
	require.NoError(t, err)
	require.Zero(t, backoff)
	aList := unstructured.UnstructuredList{}
	aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1alpha1", Kind: "KymaList"})

	operation.K8sClient.List(context.Background(), &aList)

	assert.Equal(t, 1, len(aList.Items))

}

func fixOperationForApplyKymaResource(t *testing.T) internal.Operation {
	operation := fixture.FixOperation("op-id", "inst-id", internal.OperationTypeProvision)
	operation.KymaTemplate = `
apiVersion: operator.kyma-project.io/v1alpha1
kind: Kyma
metadata:
    name: my-kyma
    namespace: kyma-system
spec:
    sync:
        strategy: secret
    channel: stable
    modules: []
`
	if len(os.Getenv("KUBECONFIG")) > 0 && strings.ToLower(os.Getenv("USE_CKUBECONFIG")) == "true" {
		config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
		if err != nil {
			t.Fatal(err.Error())
		}

		// controller-runtime lib
		scheme.Scheme.AddKnownTypeWithName(kymaGVK, &unstructured.Unstructured{})

		operation.K8sClient, err = client.New(config, client.Options{})
		if err != nil {
			t.Fatal(err.Error())
		}
	} else {
		operation.K8sClient = fake.NewClientBuilder().Build()
	}

	return operation
}
