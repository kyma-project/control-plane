package provisioning

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/steps"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

/*
Running tests with real K8S cluster instead of fake client.

k3d cluster create

kubectl create ns kyma-system

kubectl apply -f https://raw.githubusercontent.com/kyma-project/lifecycle-manager/main/operator/config/crd/bases/operator.kyma-project.io_kymas.yaml

k3d kubeconfig get --all > kubeconfig.yaml

export KUBECONFIG=kubeconfig.yaml

USE_KUBECONFIG=true go test -run TestCreatingKymaResource

kubectl get kymas -o yaml -n kyma-system
*/

func TestCreatingKymaResource(t *testing.T) {
	// given
	operation, cli := fixOperationForApplyKymaResource(t)
	storage := storage.NewMemoryStorage()
	storage.Operations().InsertOperation(operation)
	svc := NewApplyKymaStep(storage.Operations(), cli)

	// when
	_, backoff, err := svc.Run(operation, logrus.New())

	// then
	require.NoError(t, err)
	require.Zero(t, backoff)
	aList := unstructured.UnstructuredList{}
	aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1alpha1", Kind: "KymaList"})

	cli.List(context.Background(), &aList)
	assert.Equal(t, 1, len(aList.Items))
	assertLabelsExists(t, aList.Items[0])

	svc.Run(operation, logrus.New())
}

func TestCreatingKymaResource_UseNamespaceFromTimeOfCreationNotTemplate(t *testing.T) {
	// given
	operation, cli := fixOperationForApplyKymaResource(t)
	operation.KymaResourceNamespace = "namespace-in-time-of-creation"
	storage := storage.NewMemoryStorage()
	storage.Operations().InsertOperation(operation)
	svc := NewApplyKymaStep(storage.Operations(), cli)

	// when
	_, backoff, err := svc.Run(operation, logrus.New())

	// then
	require.NoError(t, err)
	require.Zero(t, backoff)
	aList := unstructured.UnstructuredList{}
	aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1alpha1", Kind: "KymaList"})

	cli.List(context.Background(), &aList)
	assert.Equal(t, 1, len(aList.Items))
	assertLabelsExists(t, aList.Items[0])

	svc.Run(operation, logrus.New())
	assert.Equal(t, "namespace-in-time-of-creation", operation.KymaResourceNamespace)
}

func TestUpdatingKymaResourceIfExists(t *testing.T) {
	// given
	operation, cli := fixOperationForApplyKymaResource(t)
	storage := storage.NewMemoryStorage()
	storage.Operations().InsertOperation(operation)
	svc := NewApplyKymaStep(storage.Operations(), cli)
	err := cli.Create(context.Background(), &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "operator.kyma-project.io/v1alpha1",
		"kind":       "Kyma",
		"metadata": map[string]interface{}{
			"name":      operation.RuntimeID,
			"namespace": "kyma-system",
		},
		"spec": map[string]interface{}{
			"channel": "stable",
		},
	}})
	require.NoError(t, err)

	// when
	_, backoff, err := svc.Run(operation, logrus.New())

	// then
	require.NoError(t, err)
	require.Zero(t, backoff)
	aList := unstructured.UnstructuredList{}
	aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1alpha1", Kind: "KymaList"})

	cli.List(context.Background(), &aList)
	assert.Equal(t, 1, len(aList.Items))
	assertLabelsExists(t, aList.Items[0])
}

func assertLabelsExists(t *testing.T, obj unstructured.Unstructured) {
	assert.Contains(t, obj.GetLabels(), "kyma-project.io/instance-id")
	assert.Contains(t, obj.GetLabels(), "kyma-project.io/runtime-id")
	assert.Contains(t, obj.GetLabels(), "kyma-project.io/global-account-id")
}

func fixOperationForApplyKymaResource(t *testing.T) (internal.Operation, client.Client) {
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
	var cli client.Client
	if len(os.Getenv("KUBECONFIG")) > 0 && strings.ToLower(os.Getenv("USE_KUBECONFIG")) == "true" {
		config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
		if err != nil {
			t.Fatal(err.Error())
		}
		// controller-runtime lib
		scheme.Scheme.AddKnownTypeWithName(steps.KymaResourceGroupVersionKind(), &unstructured.Unstructured{})

		cli, err = client.New(config, client.Options{})
		if err != nil {
			t.Fatal(err.Error())
		}
		fmt.Println("using kubeconfig")
	} else {
		fmt.Println("using fake client")
		cli = fake.NewClientBuilder().Build()
	}

	return operation, cli
}
