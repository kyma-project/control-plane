package hyperscaler

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_apis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SharedPool interface {
	SharedCredentialsSecretBinding(hyperscalerType Type) (*v1beta1.SecretBinding, error)
}

func NewSharedGardenerAccountPool(secretBindingsClient gardener_apis.SecretBindingInterface, shootsClient gardener_apis.ShootInterface) SharedPool {
	return &sharedAccountPool{
		secretBindingsClient: secretBindingsClient,
		shootsClient:         shootsClient,
	}
}

type sharedAccountPool struct {
	secretBindingsClient gardener_apis.SecretBindingInterface
	shootsClient         gardener_apis.ShootInterface
}

func (sp *sharedAccountPool) SharedCredentialsSecretBinding(hyperscalerType Type) (*v1beta1.SecretBinding, error) {
	labelSelector := fmt.Sprintf("shared=true,hyperscalerType=%s", hyperscalerType)
	secretBindings, err := sp.getSecretBindings(labelSelector)
	if err != nil {
		return nil, errors.Wrap(err, "getting secret binding")
	}

	return sp.getLeastUsed(secretBindings)
}

func (sp *sharedAccountPool) getSecretBindings(labelSelector string) ([]v1beta1.SecretBinding, error) {
	secretBindings, err := sp.secretBindingsClient.List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "error listing secret bindings for %s label selector", labelSelector)
	}

	if secretBindings == nil || len(secretBindings.Items) == 0 {
		return nil, errors.Errorf("sharedAccountPool error: no shared secret binding found for %s label selector", labelSelector)
	}

	return secretBindings.Items, nil
}

func (sp *sharedAccountPool) getLeastUsed(secretBindings []v1beta1.SecretBinding) (*v1beta1.SecretBinding, error) {
	usageCount := make(map[string]int, len(secretBindings))
	for _, s := range secretBindings {
		usageCount[s.Name] = 0
	}

	shoots, err := sp.shootsClient.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "error while listing Shoots")
	}

	if shoots == nil || len(shoots.Items) == 0 {
		return &secretBindings[0], nil
	}

	for _, s := range shoots.Items {
		count, found := usageCount[s.Spec.SecretBindingName]
		if !found {
			continue
		}

		usageCount[s.Spec.SecretBindingName] = count + 1
	}

	min := usageCount[secretBindings[0].Name]
	minIndex := 0

	for i, sb := range secretBindings {
		if usageCount[sb.Name] < min {
			min = usageCount[sb.Name]
			minIndex = i
		}
	}

	return &secretBindings[minIndex], nil
}
