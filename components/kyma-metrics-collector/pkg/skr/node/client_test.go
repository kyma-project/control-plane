package node

import (
	"context"
	kmccache "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/cache"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	skrcommons "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/commons"
	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"
	"github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	corev1 "k8s.io/api/core/v1"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

const (
	totalQueriesMetricFullName = "kmc_skr_query_total"
)

func TestList(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	// given
	ctx := context.Background()
	givenShootInfo := kmccache.Record{
		InstanceID:      "adccb200-6052-4192-8adf-785b8a5af306",
		RuntimeID:       "fe5ab5d6-5b0b-4b70-9644-7f89d230b516",
		SubAccountID:    "1ae0dbe1-d13d-4e39-bed4-7c83364084d5",
		GlobalAccountID: "0c22f798-e572-4fc7-a502-cd825c742ff6",
		ShootName:       "c-987654",
	}
	nodeList := kmctesting.Get3NodesWithStandardD8v3VMType()
	client, err := NewFakeClient(nodeList, givenShootInfo)
	g.Expect(err).Should(gomega.BeNil())

	// when
	gotNodeList, err := client.List(ctx)

	// then
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(len(gotNodeList.Items)).To(gomega.Equal(len(nodeList.Items)))
	g.Expect(*gotNodeList).To(gomega.Equal(*nodeList))

	// ensure metrics.
	gotMetrics, err := skrcommons.TotalQueriesMetric.GetMetricWithLabelValues(
		skrcommons.ListingNodesAction,
		strconv.FormatBool(true),
		givenShootInfo.ShootName,
		givenShootInfo.InstanceID,
		givenShootInfo.RuntimeID,
		givenShootInfo.SubAccountID,
		givenShootInfo.GlobalAccountID,
	)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(1)))

	// given - another case.
	// Delete all the nodes
	for _, node := range nodeList.Items {
		err := client.Resource.Delete(ctx, node.Name, metaV1.DeleteOptions{})
		g.Expect(err).Should(gomega.BeNil())
	}

	// when
	gotNodeList, err = client.List(ctx)

	// then
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(len(gotNodeList.Items)).To(gomega.Equal(0))
	// check if the required labels exists in the metric.
	gotMetrics, err = skrcommons.TotalQueriesMetric.GetMetricWithLabelValues(
		skrcommons.ListingNodesAction,
		strconv.FormatBool(true),
		givenShootInfo.ShootName,
		givenShootInfo.InstanceID,
		givenShootInfo.RuntimeID,
		givenShootInfo.SubAccountID,
		givenShootInfo.GlobalAccountID,
	)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(2)))
}

func NewFakeClient(nodeList *corev1.NodeList, shootInfo kmccache.Record) (*Client, error) {
	scheme, err := skrcommons.SetupScheme()
	if err != nil {
		return nil, err
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			{Group: "core", Version: "v1", Resource: "Node"}: "NodeList",
		}, nodeList)

	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient, ShootInfo: shootInfo}, nil
}
