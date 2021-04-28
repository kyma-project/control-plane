package secret

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/commons"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGet(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	ctx := context.Background()
	secret := &corev1.Secret{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "foo-shoot.kubeconfig",
			Namespace: "default",
		},
		StringData: map[string]string{
			"foo": "bar",
		},
	}
	nsResourceClient, err := NewFakeClient(secret)
	g.Expect(err).Should(gomega.BeNil())
	client := Client{ResourceClient: nsResourceClient}

	gotSecret, err := client.Get(ctx, "foo-shoot")
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(*gotSecret).To(gomega.Equal(*secret))

	gotSecret, err = client.Get(ctx, "doesnotexist-shoot")
	g.Expect(err).ShouldNot(gomega.BeNil())
	g.Expect(k8sErrors.IsNotFound(err)).To(gomega.BeTrue())
}

func NewFakeClient(secret *corev1.Secret) (dynamic.ResourceInterface, error) {
	scheme, err := commons.SetupSchemeOrDie()
	if err != nil {
		return nil, err
	}
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(secret)
	if err != nil {
		return nil, err
	}
	secretUnstructured := &unstructured.Unstructured{Object: unstructuredMap}
	secretUnstructured.SetGroupVersionKind(GroupVersionKind())

	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, secretUnstructured)
	nsResourceClient := dynamicClient.Resource(GroupVersionResource()).Namespace("default")

	return nsResourceClient, nil
}
