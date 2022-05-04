package update

import (
	"fmt"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var sb = []byte(`
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: servicebindings.services.cloud.sap.com
spec:
  group: services.cloud.sap.com
  names:
    kind: ServiceBinding
    listKind: ServiceBindingList
    plural: servicebindings
    singular: servicebinding
  scope: Namespaced
`)

var sbByReconciler = []byte(`
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: servicebindings.services.cloud.sap.com
  labels:
    reconciler.kyma-project.io/managed-by: reconciler
spec:
  group: services.cloud.sap.com
  names:
    kind: ServiceBinding
    listKind: ServiceBindingList
    plural: servicebindings
    singular: servicebinding
  scope: Namespaced
`)

var dummy = []byte(`
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: some-other-crd
spec:
  group: my.crd..group.sap.com
  names:
    kind: ServiceBinding
    listKind: ServiceBindingList
    plural: servicebindings
    singular: servicebinding
  scope: Namespaced
`)

func TestNewBTPOperatorCheckStep_CRDExists(t *testing.T) {
	st := storage.NewMemoryStorage()
	svc := NewBTPOperatorCheckStep(st.Operations())
	scheme := runtime.NewScheme()
	err := apiextensionsv1.AddToScheme(scheme)
	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()
	obj, gvk, err := decoder.Decode(sb, nil, nil)
	fmt.Println(gvk)
	require.NoError(t, err)
	cli := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(obj).Build()

	// when
	exists, err := svc.CRDsInstalledByUser(cli)
	require.NoError(t, err)

	// then
	assert.True(t, exists)
}

func TestNewBTPOperatorCheckStep_CRDNotExists(t *testing.T) {
	st := storage.NewMemoryStorage()
	svc := NewBTPOperatorCheckStep(st.Operations())
	scheme := runtime.NewScheme()
	err := apiextensionsv1.AddToScheme(scheme)
	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()
	obj, gvk, err := decoder.Decode(dummy, nil, nil)
	fmt.Println(gvk)
	require.NoError(t, err)
	cli := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(obj).Build()

	// when
	exists, err := svc.CRDsInstalledByUser(cli)
	require.NoError(t, err)

	// then
	assert.False(t, exists)
}

func TestNewBTPOperatorCheckStep_CRDManagedByREconciler(t *testing.T) {
	st := storage.NewMemoryStorage()
	svc := NewBTPOperatorCheckStep(st.Operations())
	scheme := runtime.NewScheme()
	err := apiextensionsv1.AddToScheme(scheme)
	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()
	obj, gvk, err := decoder.Decode(sbByReconciler, nil, nil)
	fmt.Println(gvk)
	require.NoError(t, err)
	cli := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(obj).Build()

	// when
	exists, err := svc.CRDsInstalledByUser(cli)
	require.NoError(t, err)

	// then
	assert.False(t, exists)
}
