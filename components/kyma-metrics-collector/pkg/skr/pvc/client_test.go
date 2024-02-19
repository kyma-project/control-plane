package pvc

import (
	"context"
	kmccache "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/cache"
	"sort"
	"strconv"
	"testing"

	"github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	skrcommons "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/commons"
	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"
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

	pvcList := kmctesting.GetPVCs()
	client, err := NewFakeClient(pvcList, givenShootInfo)
	g.Expect(err).Should(gomega.BeNil())

	// when
	gotPVCList, err := client.List(ctx)

	// then
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(len(gotPVCList.Items)).To(gomega.Equal(len(pvcList.Items)))
	sort.Slice(gotPVCList.Items, func(i, j int) bool {
		return gotPVCList.Items[i].Name < gotPVCList.Items[j].Name
	})
	g.Expect(*gotPVCList).To(gomega.Equal(*pvcList))

	// ensure metrics.
	gotMetrics, err := skrcommons.TotalQueriesMetric.GetMetricWithLabelValues(
		skrcommons.ListingPVCsAction,
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
	// Delete all the pvcs
	for _, pvc := range pvcList.Items {
		err := client.Resource.Namespace(pvc.Namespace).Delete(ctx, pvc.Name, metaV1.DeleteOptions{})
		g.Expect(err).Should(gomega.BeNil())
	}

	// when
	gotPVCList, err = client.List(ctx)

	// then
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(len(gotPVCList.Items)).To(gomega.Equal(0))
	// ensure metrics.
	gotMetrics, err = skrcommons.TotalQueriesMetric.GetMetricWithLabelValues(
		skrcommons.ListingPVCsAction,
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

func NewFakeClient(pvcList *corev1.PersistentVolumeClaimList, shootInfo kmccache.Record) (*Client, error) {
	scheme, err := skrcommons.SetupScheme()
	if err != nil {
		return nil, err
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			{Group: "core", Version: "v1", Resource: "PersistentVolumeClaim"}: "PersistentVolumeClaimList",
		}, pvcList)

	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient, ShootInfo: shootInfo}, nil
}
