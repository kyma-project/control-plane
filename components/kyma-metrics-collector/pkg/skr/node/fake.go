package node

import (
	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/commons"
	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

type FakeNodeClient struct{}

func (fakeNodeClient FakeNodeClient) NewClient(string) (*Client, error) {
	nodeList := kmctesting.Get3NodesWithStandardD8v3VMType()
	scheme, err := commons.SetupSchemeOrDie()
	if err != nil {
		return nil, err
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, nodeList)
	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient}, nil
}
