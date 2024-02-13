package node

import (
	k8scommons "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/k8s/commons"
	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

type FakeNodeClient struct{}

func (fakeNodeClient FakeNodeClient) NewClient(string) (*Client, error) {
	nodeList := kmctesting.Get3NodesWithStandardD8v3VMType()
	scheme, err := k8scommons.SetupScheme()
	if err != nil {
		return nil, err
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			{Group: "core", Version: "v1", Resource: "Node"}: "NodeList",
		}, nodeList)

	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient}, nil
}
