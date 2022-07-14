package shoot

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	gardenerv1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/options"
	gardenercommons "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/commons"
)

type Client struct {
	ResourceClient dynamic.ResourceInterface
}

func NewClient(opts *options.Options) (*Client, error) {
	k8sConfig := gardenercommons.GetGardenerKubeconfig(opts.GardenerSecretPath)
	client, err := k8sConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	restConfig := dynamic.ConfigFor(client)
	dynClient := dynamic.NewForConfigOrDie(restConfig)
	resourceClient := dynClient.Resource(GroupVersionResource()).Namespace(opts.GardenerNamespace)
	return &Client{ResourceClient: resourceClient}, nil
}

func (c Client) Get(ctx context.Context, shootName string) (*gardenerv1beta1.Shoot, error) {
	unstructuredShoot, err := c.ResourceClient.Get(ctx, shootName, metaV1.GetOptions{})

	if err == nil {
		gardenercommons.TotalCalls.WithLabelValues(gardenercommons.SuccessStatusLabel, shootName, gardenercommons.SuccessGettingShootLabel).Inc()
		return convertRuntimeObjToShoot(unstructuredShoot)
	}

	if !errors.IsNotFound(err) {
		gardenercommons.TotalCalls.WithLabelValues(gardenercommons.FailureStatusLabel, shootName, gardenercommons.FailedGettingShootLabel).Inc()
	}

	return nil, err
}

func convertRuntimeObjToShoot(shootUnstructured *unstructured.Unstructured) (*gardenerv1beta1.Shoot, error) {
	shoot := new(gardenerv1beta1.Shoot)
	err := k8sruntime.DefaultUnstructuredConverter.FromUnstructured(shootUnstructured.Object, shoot)
	if err != nil {
		return nil, err
	}
	return shoot, nil
}

func GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Version:  gardenerv1beta1.SchemeGroupVersion.Version,
		Group:    gardenerv1beta1.SchemeGroupVersion.Group,
		Resource: "shoots",
	}
}

func GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Version: gardenerv1beta1.SchemeGroupVersion.Version,
		Group:   gardenerv1beta1.SchemeGroupVersion.Group,
		Kind:    "Shoot",
	}
}
