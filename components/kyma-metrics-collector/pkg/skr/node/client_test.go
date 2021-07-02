package node

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gardenercommons "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/commons"
	skrcommons "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/commons"
	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"
	"github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestList(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	ctx := context.Background()

	nodeList := kmctesting.Get3NodesWithStandardD8v3VMType()
	client, err := NewFakeClient(nodeList)
	g.Expect(err).Should(gomega.BeNil())

	gotNodeList, err := client.List(ctx)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(len(gotNodeList.Items)).To(gomega.Equal(len(nodeList.Items)))
	g.Expect(*gotNodeList).To(gomega.Equal(*nodeList))
	// Tests metric
	metricName := "kmc_skr_calls_total"
	g.Expect(testutil.CollectAndCount(skrcommons.TotalCalls, metricName)).Should(gomega.Equal(2))
	callsSuccess, err := skrcommons.TotalCalls.GetMetricWithLabelValues(skrcommons.SuccessStatusLabel, skrcommons.SuccessListingNodesLabel)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(callsSuccess)).Should(gomega.Equal(float64(1)))
	callsTotal, err := skrcommons.TotalCalls.GetMetricWithLabelValues(skrcommons.CallsTotalLabel, skrcommons.ListingNodesLabel)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(callsTotal)).Should(gomega.Equal(float64(1)))

	// Delete all the nodes
	for _, node := range nodeList.Items {
		err := client.Resource.Delete(ctx, node.Name, metaV1.DeleteOptions{})
		g.Expect(err).Should(gomega.BeNil())
	}

	gotNodeList, err = client.List(ctx)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(len(gotNodeList.Items)).To(gomega.Equal(0))
	// Tests metric
	g.Expect(testutil.CollectAndCount(skrcommons.TotalCalls, metricName)).Should(gomega.Equal(2))
	callsSuccess, err = skrcommons.TotalCalls.GetMetricWithLabelValues(skrcommons.SuccessStatusLabel, skrcommons.SuccessListingNodesLabel)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(callsSuccess)).Should(gomega.Equal(float64(2)))
	callsTotal, err = skrcommons.TotalCalls.GetMetricWithLabelValues(skrcommons.CallsTotalLabel, skrcommons.ListingNodesLabel)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(callsTotal)).Should(gomega.Equal(float64(2)))
}

func NewFakeClient(nodeList *corev1.NodeList) (*Client, error) {
	scheme, err := gardenercommons.SetupSchemeOrDie()
	if err != nil {
		return nil, err
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, nodeList)
	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient}, nil
}
