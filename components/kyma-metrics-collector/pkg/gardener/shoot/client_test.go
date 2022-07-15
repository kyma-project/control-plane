package shoot

import (
	"context"
	"testing"

	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/onsi/gomega"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"

	gardenerv1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenercommons "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/commons"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestGet(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	ctx := context.Background()

	existingShoot := "foo-shoot"
	shoot := kmctesting.GetShoot(existingShoot, kmctesting.WithVMSpecs)
	nsResourceClient, err := NewFakeClient(shoot)
	g.Expect(err).Should(gomega.BeNil())
	client := Client{ResourceClient: nsResourceClient}

	gotShoot, err := client.Get(ctx, existingShoot)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(*gotShoot).To(gomega.Equal(*shoot))
	// Tests metric
	metricName := "kmc_gardener_calls_total"
	g.Expect(testutil.CollectAndCount(gardenercommons.TotalCalls, metricName)).Should(gomega.Equal(1))
	callsSuccess, err := gardenercommons.TotalCalls.GetMetricWithLabelValues(gardenercommons.SuccessStatusLabel, existingShoot, gardenercommons.SuccessGettingShootLabel)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(callsSuccess)).Should(gomega.Equal(float64(1)))

	nonexistentShoot := "doesnotexist-shoot"
	_, err = client.Get(ctx, nonexistentShoot)
	g.Expect(err).ShouldNot(gomega.BeNil())
	g.Expect(k8sErrors.IsNotFound(err)).To(gomega.BeTrue())
	// Test metric
	g.Expect(testutil.CollectAndCount(gardenercommons.TotalCalls, metricName)).Should(gomega.Equal(1))
	callsFailure, err := gardenercommons.TotalCalls.GetMetricWithLabelValues(gardenercommons.FailureStatusLabel, nonexistentShoot, gardenercommons.FailedGettingShootLabel)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(callsFailure)).Should(gomega.Equal(float64(0)))
}

func NewFakeClient(shoot *gardenerv1beta1.Shoot) (dynamic.ResourceInterface, error) {
	scheme, err := gardenercommons.SetupSchemeOrDie()
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
