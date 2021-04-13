package shoot

import (
	"context"
	"testing"

	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"

	"github.com/onsi/gomega"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"

	gardenerv1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/commons"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestGet(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	ctx := context.Background()

	shoot := kmctesting.GetShoot("foo-shoot", kmctesting.WithVMSpecs)
	nsResourceClient, err := NewFakeClient(shoot)
	g.Expect(err).Should(gomega.BeNil())
	client := Client{ResourceClient: nsResourceClient}

	gotShoot, err := client.Get(ctx, "foo-shoot")
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(*gotShoot).To(gomega.Equal(*shoot))

	gotShoot, err = client.Get(ctx, "doesnotexist-shoot")
	g.Expect(err).ShouldNot(gomega.BeNil())
	g.Expect(k8sErrors.IsNotFound(err)).To(gomega.BeTrue())
}

func NewFakeClient(shoot *gardenerv1beta1.Shoot) (dynamic.ResourceInterface, error) {
	scheme, err := commons.SetupSchemeOrDie()
	if err != nil {
		return nil, err
	}
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(shoot)
	if err != nil {
		return nil, err
	}
	shootUnstructured := &unstructured.Unstructured{Object: unstructuredMap}
	shootUnstructured.SetGroupVersionKind(GroupVersionKind())

	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, shootUnstructured)
	nsResourceClient := dynamicClient.Resource(GroupVersionResource()).Namespace("default")

	return nsResourceClient, nil
}
