package deprovisioning

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var siCRD = []byte(`
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: serviceinstances.services.cloud.sap.com
spec:
  group: services.cloud.sap.com
  names:
    kind: ServiceInstance
    listKind: ServiceInstanceList
    plural: serviceinstances
    singular: serviceinstance
  scope: Namespaced
`)

func TestRemoveServiceInstanceStep(t *testing.T) {
	t.Run("should remove uaa-issuer service instance (service catalog)", func(t *testing.T) {
		// given
		log := logrus.New()
		ms := storage.NewMemoryStorage()
		si := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "servicecatalog.k8s.io/v1beta1",
			"kind":       "ServiceInstance",
			"metadata": map[string]interface{}{
				"name":      "uaa-issuer",
				"namespace": "kyma-system",
			},
		}}

		scheme := runtime.NewScheme()
		err := apiextensionsv1.AddToScheme(scheme)
		decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()
		obj, gvk, err := decoder.Decode(siCRD, nil, nil)
		fmt.Println(gvk)
		require.NoError(t, err)

		k8sCli := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(obj).Build()
		err = k8sCli.Create(context.TODO(), si)
		require.NoError(t, err)

		op := fixture.FixDeprovisioningOperation(fixOperationID, fixInstanceID)
		op.State = "in progress"
		op.K8sClient = k8sCli

		step := NewRemoveServiceInstanceStep(ms.Operations())

		// when
		entry := log.WithFields(logrus.Fields{"step": "TEST"})
		_, _, err = step.Run(op, entry)

		// then
		assert.NoError(t, err)

		// given
		emptySI := &unstructured.Unstructured{}
		emptySI.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "servicecatalog.k8s.io",
			Version: "v1beta1",
			Kind:    "ServiceInstance",
		})

		// when
		err = k8sCli.Get(context.TODO(), client.ObjectKey{
			Namespace: serviceInstanceNamespace,
			Name:      serviceInstanceName,
		}, emptySI)

		// then
		assert.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("should remove uaa-issuer service instance (btp operator)", func(t *testing.T) {
		// given
		log := logrus.New()
		ms := storage.NewMemoryStorage()
		si := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "services.cloud.sap.com/v1",
			"kind":       "ServiceInstance",
			"metadata": map[string]interface{}{
				"name":      "uaa-issuer",
				"namespace": "kyma-system",
			},
		}}

		scheme := runtime.NewScheme()
		err := apiextensionsv1.AddToScheme(scheme)
		decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()
		obj, gvk, err := decoder.Decode(siCRD, nil, nil)
		fmt.Println(gvk)
		require.NoError(t, err)

		k8sCli := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(obj).Build()
		err = k8sCli.Create(context.TODO(), si)
		require.NoError(t, err)

		op := fixture.FixDeprovisioningOperation(fixOperationID, fixInstanceID)
		op.State = "in progress"
		op.K8sClient = k8sCli

		step := NewRemoveServiceInstanceStep(ms.Operations())

		// when
		entry := log.WithFields(logrus.Fields{"step": "TEST"})
		_, _, err = step.Run(op, entry)

		// then
		assert.NoError(t, err)

		// given
		emptySI := &unstructured.Unstructured{}
		emptySI.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "services.cloud.sap.com",
			Version: "v1",
			Kind:    "ServiceInstance",
		})

		// when
		err = k8sCli.Get(context.TODO(), client.ObjectKey{
			Namespace: serviceInstanceNamespace,
			Name:      serviceInstanceName,
		}, emptySI)

		// then
		assert.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("should not find uaa-issuer service instance and set flag to true", func(t *testing.T) {
		// given
		log := logrus.New()
		ms := storage.NewMemoryStorage()

		scheme := runtime.NewScheme()
		err := apiextensionsv1.AddToScheme(scheme)
		decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()
		obj, gvk, err := decoder.Decode(siCRD, nil, nil)
		fmt.Println(gvk)
		require.NoError(t, err)

		k8sCli := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(obj).Build()
		require.NoError(t, err)

		op := fixture.FixDeprovisioningOperation(fixOperationID, fixInstanceID)
		op.State = "in progress"
		op.K8sClient = k8sCli

		step := NewRemoveServiceInstanceStep(ms.Operations())

		// when
		entry := log.WithFields(logrus.Fields{"step": "TEST"})
		op, _, err = step.Run(op, entry)

		// then
		assert.NoError(t, err)
		assert.True(t, op.IsServiceInstanceDeleted)
	})

	t.Run("should return step repeat time with 20 sec", func(t *testing.T) {
		// given
		log := logrus.New()
		ms := storage.NewMemoryStorage()
		si := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "services.cloud.sap.com/v1",
			"kind":       "ServiceInstance",
			"metadata": map[string]interface{}{
				"name":      "uaa-issuer",
				"namespace": "kyma-system",
			},
		}}

		k8sCli := fakeK8sClient{}
		err := k8sCli.Create(context.TODO(), si)
		require.NoError(t, err)

		op := fixture.FixDeprovisioningOperation(fixOperationID, fixInstanceID)
		op.State = "in progress"
		op.K8sClient = &k8sCli

		step := NewRemoveServiceInstanceStep(ms.Operations())

		// when
		entry := log.WithFields(logrus.Fields{"step": "TEST"})
		op, repeat, err := step.Run(op, entry)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Second*20, repeat)
		assert.False(t, op.IsServiceInstanceDeleted)
	})
}

type fakeK8sClient struct {
	obj *unstructured.Unstructured
}

func (f *fakeK8sClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	obj = f.obj
	return nil
}

func (f *fakeK8sClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return nil
}

func (f *fakeK8sClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	f.obj = new(unstructured.Unstructured)
	f.obj.SetName(obj.GetName())
	f.obj.SetNamespace(obj.GetNamespace())
	return nil
}

func (f *fakeK8sClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return k8serrors.NewInternalError(errors.New("test error"))
}

func (f *fakeK8sClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return nil
}

func (f *fakeK8sClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}

func (f *fakeK8sClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return nil
}

func (f *fakeK8sClient) Status() client.StatusWriter {
	return nil
}

func (f *fakeK8sClient) Scheme() *runtime.Scheme {
	return nil
}

func (f *fakeK8sClient) RESTMapper() meta.RESTMapper {
	return nil
}
