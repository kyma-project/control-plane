package hyperscaler

import (
	"context"
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

type SharedPool interface {
	SharedCredentialsSecretBinding(hyperscalerType Type) (*gardener.SecretBinding, error)
}

func NewSharedGardenerAccountPool(gardenerClient dynamic.Interface, gardenerNamespace string) SharedPool {
	return &sharedAccountPool{
		gardenerClient: gardenerClient,
		namespace:      gardenerNamespace,
	}
}

type sharedAccountPool struct {
	gardenerClient dynamic.Interface
	namespace      string
}

func (sp *sharedAccountPool) SharedCredentialsSecretBinding(hyperscalerType Type) (*gardener.SecretBinding, error) {
	labelSelector := fmt.Sprintf("shared=true,hyperscalerType=%s", hyperscalerType)
	secretBindings, err := sp.getSecretBindings(labelSelector)
	if err != nil {
		return nil, errors.Wrap(err, "getting secret binding")
	}

	return sp.getLeastUsed(secretBindings)
}

func (sp *sharedAccountPool) getSecretBindings(labelSelector string) ([]unstructured.Unstructured, error) {
	secretBindings, err := sp.gardenerClient.Resource(gardener.SecretBindingResource).Namespace(sp.namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "error listing secret bindings for %s label selector", labelSelector)
	}

	if secretBindings == nil || len(secretBindings.Items) == 0 {
		return nil, errors.Errorf("sharedAccountPool error: no shared secret binding found for %s label selector, namespace %s", labelSelector, sp.namespace)
	}

	return secretBindings.Items, nil
}

func (sp *sharedAccountPool) getLeastUsed(secretBindings []unstructured.Unstructured) (*gardener.SecretBinding, error) {
	usageCount := make(map[string]int, len(secretBindings))
	for _, s := range secretBindings {
		usageCount[s.GetName()] = 0
	}

	shoots, err := sp.gardenerClient.Resource(gardener.ShootResource).Namespace(sp.namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "error while listing Shoots")
	}

	if shoots == nil || len(shoots.Items) == 0 {
		return &gardener.SecretBinding{secretBindings[0]}, nil
	}

	for _, shoot := range shoots.Items {
		s := gardener.Shoot{shoot}
		count, found := usageCount[s.GetSpecSecretBindingName()]
		if !found {
			continue
		}

		usageCount[s.GetSpecSecretBindingName()] = count + 1
	}

	min := usageCount[secretBindings[0].GetName()]
	minIndex := 0

	for i, sb := range secretBindings {
		if usageCount[sb.GetName()] < min {
			min = usageCount[sb.GetName()]
			minIndex = i
		}
	}

	return &gardener.SecretBinding{secretBindings[minIndex]}, nil
}
