package node

import (
	"github.com/kyma-incubator/metris/pkg/gardener/commons"
	metristesting "github.com/kyma-incubator/metris/pkg/testing"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

type FakeNodeClient struct{}

func (fakeNodeClient FakeNodeClient) NewClient(string) (*Client, error) {
	nodeList := metristesting.Get3NodesWithStandardD8v3VMType()
	scheme, err := commons.SetupSchemeOrDie()
	if err != nil {
		return nil, err
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, nodeList)
	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient}, nil
}
