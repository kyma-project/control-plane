package pvc

import (
	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/commons"
	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

type FakePVCClient struct{}

func (fakePVCClient FakePVCClient) NewClient(string) (*Client, error) {
	pvcList := kmctesting.GetPVCs()
	scheme, err := commons.SetupSchemeOrDie()
	if err != nil {
		return nil, err
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, pvcList)
	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient}, nil
}
