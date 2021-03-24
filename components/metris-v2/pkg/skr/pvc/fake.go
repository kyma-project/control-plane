package pvc

import (
	"github.com/kyma-incubator/metris/pkg/gardener/commons"
	metristesting "github.com/kyma-incubator/metris/pkg/testing"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

type FakePVCClient struct{}

func (fakePVCClient FakePVCClient) NewClient(string) (*Client, error) {
	pvcList := metristesting.GetPVCs()
	scheme, err := commons.SetupSchemeOrDie()
	if err != nil {
		return nil, err
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, pvcList)
	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient}, nil
}
