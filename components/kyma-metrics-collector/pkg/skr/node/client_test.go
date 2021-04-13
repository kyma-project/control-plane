package node

import (
	"context"
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

	nodeList := kmctesting.Get3NodesWithStandardD8v3VMType()
	client, err := NewFakeClient(nodeList)
	g.Expect(err).Should(gomega.BeNil())

	gotNodeList, err := client.List(ctx)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(len(gotNodeList.Items)).To(gomega.Equal(len(nodeList.Items)))
	g.Expect(*gotNodeList).To(gomega.Equal(*nodeList))

	// Delete all the nodes
	for _, node := range nodeList.Items {
		err := client.Resource.Delete(ctx, node.Name, metaV1.DeleteOptions{})
		g.Expect(err).Should(gomega.BeNil())
	}

	gotNodeList, err = client.List(ctx)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(len(gotNodeList.Items)).To(gomega.Equal(0))
}

func NewFakeClient(nodeList *corev1.NodeList) (*Client, error) {
	scheme, err := commons.SetupSchemeOrDie()
	if err != nil {
		return nil, err
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, nodeList)
	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient}, nil
}
