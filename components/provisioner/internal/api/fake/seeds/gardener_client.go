package seeds

import (
	"context"
	"testing"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_apis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

func NewFakeSeedsInterface(t *testing.T, config *rest.Config) gardener_apis.SeedInterface {
	dynamicClient, err := dynamic.NewForConfig(config)
	require.NoError(t, err)

	resourceInterface := dynamicClient.Resource(gardener_types.SchemeGroupVersion.WithResource("seeds"))
	return &fakeSeedsInterface{
		client: resourceInterface,
	}
}

type fakeSeedsInterface struct {
	client dynamic.ResourceInterface
}

func (f fakeSeedsInterface) Create(ctx context.Context, Seed *gardener_types.Seed, options metav1.CreateOptions) (*gardener_types.Seed, error) {
	addTypeMeta(Seed)

	Seed.SetFinalizers([]string{"finalizer"})

	unstructuredSeed, err := toUnstructured(Seed)
	if err != nil {
		return nil, err
	}

	create, err := f.client.Create(ctx, unstructuredSeed, options)
	if err != nil {
		return nil, err
	}

	return fromUnstructured(create)
}

func (f *fakeSeedsInterface) Update(ctx context.Context, Seed *gardener_types.Seed, options metav1.UpdateOptions) (*gardener_types.Seed, error) {
	obj, err := toUnstructured(Seed)

	if err != nil {
		return nil, err
	}
	updated, err := f.client.Update(context.Background(), obj, options)
	if err != nil {
		return nil, err
	}

	return fromUnstructured(updated)
}

func (f *fakeSeedsInterface) UpdateStatus(_ context.Context, _ *gardener_types.Seed, _ metav1.UpdateOptions) (*gardener_types.Seed, error) {
	return nil, nil
}

func (f *fakeSeedsInterface) Delete(ctx context.Context, name string, options metav1.DeleteOptions) error {
	return f.client.Delete(ctx, name, options)
}

func (f *fakeSeedsInterface) DeleteCollection(_ context.Context, _ metav1.DeleteOptions, _ metav1.ListOptions) error {
	return nil
}

func (f *fakeSeedsInterface) Get(ctx context.Context, name string, options metav1.GetOptions) (*gardener_types.Seed, error) {
	obj, err := f.client.Get(ctx, name, options)
	if err != nil {
		return nil, err
	}

	return fromUnstructured(obj)
}
func (f *fakeSeedsInterface) List(ctx context.Context, options metav1.ListOptions) (*gardener_types.SeedList, error) {
	list, err := f.client.List(ctx, options)
	if err != nil {
		return nil, err
	}

	return listFromUnstructured(list)
}
func (f *fakeSeedsInterface) Watch(_ context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	return nil, nil
}
func (f *fakeSeedsInterface) Patch(_ context.Context, _ string, _ types.PatchType, _ []byte, _ metav1.PatchOptions, _ ...string) (result *gardener_types.Seed, err error) {
	return nil, nil
}

func addTypeMeta(Seed *gardener_types.Seed) {
	Seed.TypeMeta = metav1.TypeMeta{
		Kind:       "Seed",
		APIVersion: "core.gardener.cloud/v1beta1",
	}
}

func toUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	object, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)

	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: object}, nil
}

func fromUnstructured(object *unstructured.Unstructured) (*gardener_types.Seed, error) {
	var newSeed gardener_types.Seed

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &newSeed)
	if err != nil {
		return nil, err
	}

	return &newSeed, err
}

func listFromUnstructured(list *unstructured.UnstructuredList) (*gardener_types.SeedList, error) {
	SeedList := &gardener_types.SeedList{
		Items: []gardener_types.Seed{},
	}

	for _, obj := range list.Items {
		Seed, err := fromUnstructured(&obj)
		if err != nil {
			return &gardener_types.SeedList{}, err
		}
		SeedList.Items = append(SeedList.Items, *Seed)
	}
	return SeedList, nil
}
