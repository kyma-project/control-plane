package pvc

import (
	"fmt"
	kmccache "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/cache"
	skrcommons "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/commons"
	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

type FakePVCClient struct{}

func (fakePVCClient FakePVCClient) NewClient(record kmccache.Record) (*Client, error) {
	// define failure scenario.
	if record.KubeConfig == "invalid" {
		return nil, fmt.Errorf("failed to create client")
	}

	// setup fake client with PVCs.
	pvcList := kmctesting.GetPVCs()
	scheme, err := skrcommons.SetupScheme()
	if err != nil {
		return nil, err
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			{Group: "core", Version: "v1", Resource: "PersistentVolumeClaim"}: "PersistentVolumeClaimList",
		}, pvcList)

	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient}, nil
}
