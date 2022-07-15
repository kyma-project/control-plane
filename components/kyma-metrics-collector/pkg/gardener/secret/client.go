package secret

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/options"
	gardenercommons "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/commons"
)

type Client struct {
	ResourceClient dynamic.ResourceInterface
}

func NewClient(opts *options.Options) (*Client, error) {
	k8sConfig := gardenercommons.GetGardenerKubeconfig(opts.GardenerSecretPath)
	clientCfg, err := k8sConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	restConfig := dynamic.ConfigFor(clientCfg)
	dynClient := dynamic.NewForConfigOrDie(restConfig)
	resourceClient := dynClient.Resource(GroupVersionResource()).Namespace(opts.GardenerNamespace)
	return &Client{ResourceClient: resourceClient}, nil
}

func (c Client) Get(ctx context.Context, shootName string) (*corev1.Secret, error) {
	shootKubeconfigName := fmt.Sprintf("%s.kubeconfig", shootName)
	unstructuredSecret, err := c.ResourceClient.Get(ctx, shootKubeconfigName, metaV1.GetOptions{})

	if err == nil {
		gardenercommons.TotalCalls.WithLabelValues(gardenercommons.SuccessStatusLabel, shootName, gardenercommons.SuccessGettingSecretLabel).Inc()
		return convertRuntimeObjToSecret(unstructuredSecret)
	}

	if !errors.IsNotFound(err) {
		gardenercommons.TotalCalls.WithLabelValues(gardenercommons.FailureStatusLabel, shootName, gardenercommons.FailedGettingSecretLabel).Inc()
	}

	return nil, err
}

func convertRuntimeObjToSecret(unstructuredSecret *unstructured.Unstructured) (*corev1.Secret, error) {
	secret := new(corev1.Secret)
	err := k8sruntime.DefaultUnstructuredConverter.FromUnstructured(unstructuredSecret.Object, secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Version:  corev1.SchemeGroupVersion.Version,
		Group:    corev1.SchemeGroupVersion.Group,
		Resource: "secrets",
	}
}

func GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Version: corev1.SchemeGroupVersion.Version,
		Group:   corev1.SchemeGroupVersion.Group,
		Kind:    "Secret",
	}
}
