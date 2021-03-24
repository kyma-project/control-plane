package svc

import (
	"github.com/kyma-incubator/metris/pkg/gardener/commons"
	metristesting "github.com/kyma-incubator/metris/pkg/testing"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

type FakeSvcClient struct{}

func (fakeSvcClient FakeSvcClient) NewClient(string) (*Client, error) {
	nodeList := metristesting.GetSvcsWithLoadBalancers()
	scheme, err := commons.SetupSchemeOrDie()
	if err != nil {
		return nil, err
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, nodeList)
	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient}, nil
}
