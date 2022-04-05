package deprovisioning

import (
	"context"
	"fmt"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
		assert.True(t, apierrors.IsNotFound(err))
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
		assert.True(t, apierrors.IsNotFound(err))
	})
}
