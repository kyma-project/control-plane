package gardener

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func NewDynamicFakeClient(objects ...runtime.Object) *fake.FakeDynamicClient {
	// dynamic fake client requirement https://github.com/kubernetes/client-go/issues/949#issuecomment-811192420
	scheme.Scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "core.gardener.cloud", Version: "v1beta1", Kind: "Shoot"}, &unstructured.Unstructured{})
	scheme.Scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "core.gardener.cloud", Version: "v1beta1", Kind: "ShootList"}, &unstructured.UnstructuredList{})
	scheme.Scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "core.gardener.cloud", Version: "v1beta1", Kind: "SecretBinding"}, &unstructured.Unstructured{})
	scheme.Scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "core.gardener.cloud", Version: "v1beta1", Kind: "SecretBindingList"}, &unstructured.UnstructuredList{})

	return fake.NewSimpleDynamicClient(scheme.Scheme, objects...)
}
