package svc

import (
	"context"
	"sort"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/commons"
	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"
	"github.com/onsi/gomega"
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
		if gotSvcList.Items[i].Name < gotSvcList.Items[j].Name {
			return true
		}
		return false
	})
	g.Expect(*gotSvcList).To(gomega.Equal(*svcList))

	// Delete all the svcs
	for _, svc := range svcList.Items {
		err := client.Resource.Namespace(svc.Namespace).Delete(ctx, svc.Name, metaV1.DeleteOptions{})
		g.Expect(err).Should(gomega.BeNil())
	}

	gotSvcList, err = client.List(ctx)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(len(gotSvcList.Items)).To(gomega.Equal(0))
}

func NewFakeClient(svcList *corev1.ServiceList) (*Client, error) {
	scheme, err := commons.SetupSchemeOrDie()
	if err != nil {
		return nil, err
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, svcList)
	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient}, nil
}
