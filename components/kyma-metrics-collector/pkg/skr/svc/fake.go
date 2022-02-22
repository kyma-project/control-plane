package svc

import (
	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/commons"
	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

type FakeSvcClient struct{}

func (fakeSvcClient FakeSvcClient) NewClient(string) (*Client, error) {
	nodeList := kmctesting.GetSvcsWithLoadBalancers()
	scheme, err := commons.SetupSchemeOrDie()
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
