package pvc

import (
	"context"
	"sort"
	"testing"

	"github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/commons"
	skrcommons "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/commons"
	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"
)

func TestList(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	ctx := context.Background()

	pvcList := kmctesting.GetPVCs()
	client, err := NewFakeClient(pvcList)
	g.Expect(err).Should(gomega.BeNil())

	gotPVCList, err := client.List(ctx)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(len(gotPVCList.Items)).To(gomega.Equal(len(pvcList.Items)))
	sort.Slice(gotPVCList.Items, func(i, j int) bool {
		return gotPVCList.Items[i].Name < gotPVCList.Items[j].Name
	})
	g.Expect(*gotPVCList).To(gomega.Equal(*pvcList))
	// Tests metric
	metricName := "kmc_skr_calls_total"
	g.Expect(testutil.CollectAndCount(skrcommons.TotalCalls, metricName)).Should(gomega.Equal(2))
	callsSuccess, err := skrcommons.TotalCalls.GetMetricWithLabelValues(skrcommons.SuccessStatusLabel, skrcommons.SuccessListingPVCLabel)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(callsSuccess)).Should(gomega.Equal(float64(1)))
	callsTotal, err := skrcommons.TotalCalls.GetMetricWithLabelValues(skrcommons.CallsTotalLabel, skrcommons.ListingPVCLabel)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(callsTotal)).Should(gomega.Equal(float64(1)))

	// Delete all the pvcs
	for _, pvc := range pvcList.Items {
		err := client.Resource.Namespace(pvc.Namespace).Delete(ctx, pvc.Name, metaV1.DeleteOptions{})
		g.Expect(err).Should(gomega.BeNil())
	}

	gotPVCList, err = client.List(ctx)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(len(gotPVCList.Items)).To(gomega.Equal(0))
	// Tests metric
	g.Expect(testutil.CollectAndCount(skrcommons.TotalCalls, metricName)).Should(gomega.Equal(2))
	callsSuccess, err = skrcommons.TotalCalls.GetMetricWithLabelValues(skrcommons.SuccessStatusLabel, skrcommons.SuccessListingPVCLabel)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(callsSuccess)).Should(gomega.Equal(float64(2)))
	callsTotal, err = skrcommons.TotalCalls.GetMetricWithLabelValues(skrcommons.CallsTotalLabel, skrcommons.ListingPVCLabel)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(callsTotal)).Should(gomega.Equal(float64(2)))
}

func NewFakeClient(pvcList *corev1.PersistentVolumeClaimList) (*Client, error) {
	scheme, err := commons.SetupSchemeOrDie()
	if err != nil {
		return nil, err
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			{Group: "core", Version: "v1", Resource: "PersistentVolumeClaim"}: "PersistentVolumeClaimList",
		}, pvcList)

	// dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, pvcList)
	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient}, nil
}
