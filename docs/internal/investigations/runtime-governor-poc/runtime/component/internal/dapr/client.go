package dapr

import (
	"errors"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	dapr "github.com/dapr/dapr/pkg/client/clientset/versioned"
	"github.com/hashicorp/go-multierror"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	daprClient *dapr.Clientset
	k8sClient  *kubernetes.Clientset
}

func NewClientOrDie(kubecfgPath string) *Client {
	var config *rest.Config
	var err error

	if kubecfgPath == "" {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubecfgPath)
	}
	if err != nil {
		panic(err)
	}

	return &Client{
		daprClient: dapr.NewForConfigOrDie(config),
		k8sClient:  kubernetes.NewForConfigOrDie(config),
	}
}

func (c Client) DeletePodsForSelector(selector, namespace string) error {
	podList, err := c.listPodsForSelector(selector, namespace)
	if err != nil {
		return err
	}

	var result error
	for _, pod := range podList.Items {
		if err := c.k8sClient.CoreV1().Pods(namespace).Delete(pod.Name, nil); err != nil {
			result = multierror.Append(result, err)
		}
	}

	return result
}

func (c Client) UpsertComponent(input *v1alpha1.Component, namespace string) (bool, error) {
	resource, err := c.daprClient.ComponentsV1alpha1().Components(namespace).Get(input.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return true, c.createComponent(input, namespace)
		}
		return false, err
	}

	reload := shouldReloadResource(input, resource)
	if !reload {
		return false, nil
	}

	err = c.daprClient.ComponentsV1alpha1().Components(namespace).Delete(input.Name, nil)
	if err != nil {
		return false, err
	}

	return reload, c.createComponent(input, namespace)
}

func (c Client) createComponent(input *v1alpha1.Component, namespace string) error {
	if input == nil {
		return errors.New("input cannot be nil")
	}

	_, err := c.daprClient.ComponentsV1alpha1().Components(namespace).Create(input)
	return err
}

func (c Client) listPodsForSelector(selector, namespace string) (*v1.PodList, error) {
	opts := metav1.ListOptions{
		LabelSelector: selector,
	}
	return c.k8sClient.CoreV1().Pods(namespace).List(opts)
}

func shouldReloadResource(newRes, oldRes *v1alpha1.Component) bool {
	reload := true
	oldVer, oldVerOk := oldRes.Annotations["kcp-res-version"]
	newVer, newVerOk := newRes.Annotations["kcp-res-version"]
	if oldVerOk && newVerOk {
		reload = oldVer != newVer
	}

	return reload
}
