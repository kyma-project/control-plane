package svc

import (
	"context"
	kmccache "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/cache"
	"sort"
	"strconv"
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

	svcList := kmctesting.GetSvcsWithLoadBalancers()
	client, err := NewFakeClient(svcList, givenShootInfo)
	g.Expect(err).Should(gomega.BeNil())

	// when
	gotSvcList, err := client.List(ctx)

	// then
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(len(gotSvcList.Items)).To(gomega.Equal(len(svcList.Items)))
	sort.Slice(gotSvcList.Items, func(i, j int) bool {
		return gotSvcList.Items[i].Name < gotSvcList.Items[j].Name
	})
	g.Expect(*gotSvcList).To(gomega.Equal(*svcList))
	// ensure metrics.
	gotMetrics, err := skrcommons.TotalQueriesMetric.GetMetricWithLabelValues(
		skrcommons.ListingSVCsAction,
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
	// Delete all the svcs
	for _, svc := range svcList.Items {
		err := client.Resource.Namespace(svc.Namespace).Delete(ctx, svc.Name, metaV1.DeleteOptions{})
		g.Expect(err).Should(gomega.BeNil())
	}

	// when
	gotSvcList, err = client.List(ctx)

	// then
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(len(gotSvcList.Items)).To(gomega.Equal(0))
	// ensure metrics.
	gotMetrics, err = skrcommons.TotalQueriesMetric.GetMetricWithLabelValues(
		skrcommons.ListingSVCsAction,
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

func NewFakeClient(svcList *corev1.ServiceList, shootInfo kmccache.Record) (*Client, error) {
	scheme, err := skrcommons.SetupScheme()
	if err != nil {
		return nil, err
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			{Group: "core", Version: "v1", Resource: "Service"}: "ServiceList",
		}, svcList)

	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient, ShootInfo: shootInfo}, nil
}
