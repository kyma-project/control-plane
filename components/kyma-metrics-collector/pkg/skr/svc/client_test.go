package svc

import (
	"context"
	"sort"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	skrcommons "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/commons"
	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"
	"github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestList(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	ctx := context.Background()

	svcList := kmctesting.GetSvcsWithLoadBalancers()
	client, err := NewFakeClient(svcList)
	g.Expect(err).Should(gomega.BeNil())

	gotSvcList, err := client.List(ctx)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(len(gotSvcList.Items)).To(gomega.Equal(len(svcList.Items)))
	sort.Slice(gotSvcList.Items, func(i, j int) bool {
		return gotSvcList.Items[i].Name < gotSvcList.Items[j].Name
	})
	g.Expect(*gotSvcList).To(gomega.Equal(*svcList))
	// Tests metric
	metricName := "kmc_skr_calls_total"
	g.Expect(testutil.CollectAndCount(skrcommons.TotalCalls, metricName)).Should(gomega.Equal(2))
	callsSuccess, err := skrcommons.TotalCalls.GetMetricWithLabelValues(skrcommons.SuccessStatusLabel, skrcommons.SuccessListingSVCLabel)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(callsSuccess)).Should(gomega.Equal(float64(1)))
	callsTotal, err := skrcommons.TotalCalls.GetMetricWithLabelValues(skrcommons.CallsTotalLabel, skrcommons.ListingSVCLabel)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(callsTotal)).Should(gomega.Equal(float64(1)))

	// Delete all the svcs
	for _, svc := range svcList.Items {
		err := client.Resource.Namespace(svc.Namespace).Delete(ctx, svc.Name, metaV1.DeleteOptions{})
		g.Expect(err).Should(gomega.BeNil())
	}

	gotSvcList, err = client.List(ctx)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(len(gotSvcList.Items)).To(gomega.Equal(0))
	// Tests metric
	g.Expect(testutil.CollectAndCount(skrcommons.TotalCalls, metricName)).Should(gomega.Equal(2))
	callsSuccess, err = skrcommons.TotalCalls.GetMetricWithLabelValues(skrcommons.SuccessStatusLabel, skrcommons.SuccessListingSVCLabel)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(callsSuccess)).Should(gomega.Equal(float64(2)))
	callsTotal, err = skrcommons.TotalCalls.GetMetricWithLabelValues(skrcommons.CallsTotalLabel, skrcommons.ListingSVCLabel)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(callsTotal)).Should(gomega.Equal(float64(2)))
}

func NewFakeClient(svcList *corev1.ServiceList) (*Client, error) {
	scheme, err := skrcommons.SetupScheme()
	if err != nil {
		return nil, err
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			{Group: "core", Version: "v1", Resource: "Service"}: "ServiceList",
		}, svcList)

	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient}, nil
}
