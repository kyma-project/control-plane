package deprovisioning

import (
	"fmt"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var si = []byte(`
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
		//given
		log := logrus.New()
		ms := storage.NewMemoryStorage()

		scheme := runtime.NewScheme()
		err := apiextensionsv1.AddToScheme(scheme)
		decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()
		obj, gvk, err := decoder.Decode(si, nil, nil)
		fmt.Println(gvk)
		require.NoError(t, err)

		k8sCli := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(obj).Build()

		op := fixture.FixDeprovisioningOperation(fixOperationID, fixInstanceID)
		op.State = "in progress"
		op.K8sClient = k8sCli

		step := NewRemoveServiceInstanceStep(ms)
	})
}
