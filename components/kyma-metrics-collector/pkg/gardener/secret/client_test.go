package secret

import (
	"context"
	"testing"

	gardenercommons "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/commons"
	"github.com/prometheus/client_golang/prometheus/testutil"
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

	existingShoot := "foo-shoot"
	gotSecret, err := client.Get(ctx, existingShoot)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(*gotSecret).To(gomega.Equal(*secret))
	// Tests metric
	metricName := "kmc_gardener_calls_total"
	g.Expect(testutil.CollectAndCount(gardenercommons.TotalCalls, metricName)).Should(gomega.Equal(1))
	callsSuccess, err := gardenercommons.TotalCalls.GetMetricWithLabelValues(gardenercommons.SuccessStatusLabel, existingShoot, gardenercommons.SuccessGettingSecretLabel)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(callsSuccess)).Should(gomega.Equal(float64(1)))

	nonexistentShoot := "doesnotexist-shoot"
	_, err = client.Get(ctx, nonexistentShoot)
	g.Expect(err).ShouldNot(gomega.BeNil())
	g.Expect(k8sErrors.IsNotFound(err)).To(gomega.BeTrue())
	// Test metric
	g.Expect(testutil.CollectAndCount(gardenercommons.TotalCalls, metricName)).Should(gomega.Equal(2))
	callsFailure, err := gardenercommons.TotalCalls.GetMetricWithLabelValues(gardenercommons.FailureStatusLabel, nonexistentShoot, gardenercommons.FailedGettingSecretLabel)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(callsFailure)).Should(gomega.Equal(float64(1)))
}

func NewFakeClient(secret *corev1.Secret) (dynamic.ResourceInterface, error) {
	scheme, err := gardenercommons.SetupSchemeOrDie()
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
