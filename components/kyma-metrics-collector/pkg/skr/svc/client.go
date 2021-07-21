package svc

import (
	"context"
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	skrcommons "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/commons"
)

type Client struct {
	Resource dynamic.NamespaceableResourceInterface
}

func (c Config) NewClient(kubeconfig string) (*Client, error) {
	restClientConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(restClientConfig)
	if err != nil {
		return nil, err
	}
	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient}, nil
}

func (c Client) List(ctx context.Context) (*corev1.ServiceList, error) {

	skrcommons.TotalCalls.WithLabelValues(skrcommons.CallsTotalLabel, skrcommons.ListingSVCLabel).Inc()
	unstructuredSvcList, err := c.Resource.Namespace(corev1.NamespaceAll).List(ctx, metaV1.ListOptions{})
	if err != nil {
		return nil, err
	}
	skrcommons.TotalCalls.WithLabelValues(skrcommons.SuccessStatusLabel, skrcommons.SuccessListingSVCLabel).Inc()
	return convertUnstructuredListToSVCList(unstructuredSvcList)
}

func convertUnstructuredListToSVCList(unstructuredSvcList *unstructured.UnstructuredList) (*corev1.ServiceList, error) {
	svcList := new(corev1.ServiceList)
	svcListBytes, err := unstructuredSvcList.MarshalJSON()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(svcListBytes, svcList)
	if err != nil {
		return nil, err
	}
	return svcList, nil
}

func GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Version:  corev1.SchemeGroupVersion.Version,
		Group:    corev1.SchemeGroupVersion.Group,
		Resource: "services",
	}
}
