package hyperscaler

import (
	"fmt"
	"sync"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_apis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
)

type SharedPool interface {
	SharedCredentials(hyperscalerType Type) (Credentials, error)
}

func NewSharedGardenerAccountPool(gardenerClusterConfig *restclient.Config, secretBindingsClient gardener_apis.SecretBindingInterface, shootsClient gardener_apis.ShootInterface) *SharedAccountPool {
	return &SharedAccountPool{
		gardenerClusterConfig: gardenerClusterConfig,
		secretBindingsClient:  secretBindingsClient,
		shootsClient:          shootsClient,
	}
}

type SharedAccountPool struct {
	gardenerClusterConfig *restclient.Config
	secretBindingsClient  gardener_apis.SecretBindingInterface
	shootsClient          gardener_apis.ShootInterface
	mux                   sync.Mutex
}

func (sp *SharedAccountPool) SharedCredentials(hyperscalerType Type) (Credentials, error) {
	labelSelector := fmt.Sprintf("shared=true,hyperscalerType=%s", hyperscalerType)
	secretBindings, err := getSecretBindings(sp.secretBindingsClient, labelSelector)
	if err != nil {
		return Credentials{}, err
	}

	secretBinding, err := sp.getLeastUsed(secretBindings)
	if err != nil {
		return Credentials{}, err
	}

	return credentialsFromBoundSecret(sp.gardenerClusterConfig, &secretBinding, hyperscalerType)
}

func getSecretBindings(secretBindingsClient gardener_apis.SecretBindingInterface, labelSelector string) ([]v1beta1.SecretBinding, error) {
	secretBindings, err := secretBindingsClient.List(metav1.ListOptions{
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

func (sp *SharedAccountPool) getLeastUsed(secretBindings []v1beta1.SecretBinding) (v1beta1.SecretBinding, error) {
	usageCount := make(map[string]int, len(secretBindings))
	for _, s := range secretBindings {
		usageCount[s.Name] = 0
	}

	shoots, err := sp.shootsClient.List(metav1.ListOptions{})
	if err != nil {
		return v1beta1.SecretBinding{}, errors.Wrap(err, "error while listing Shoots")
	}

	if shoots == nil || len(shoots.Items) == 0 {
		return secretBindings[0], nil
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

	for i, s := range secretBindings {
		if usageCount[s.Name] < min {
			min = usageCount[s.Name]
			minIndex = i
		}
	}

	return secretBindings[minIndex], nil
}
