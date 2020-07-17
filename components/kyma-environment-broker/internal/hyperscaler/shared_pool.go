package hyperscaler

import (
	"fmt"
	"github.com/pkg/errors"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	gardener_apis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
)

//
type SharedPool interface {
	SharedCredentials(hyperscalerType Type) (Credentials, error)
}

func NewSharedGardenerAccountPool(secretsClient corev1.SecretInterface, shootClient gardener_apis.ShootInterface) *SharedAccountPool {
	return &SharedAccountPool{
		secretsClient: secretsClient,
		shootsClient:  shootClient,
	}
}

type SharedAccountPool struct {
	secretsClient corev1.SecretInterface
	shootsClient gardener_apis.ShootInterface
}

func (sp *SharedAccountPool) SharedCredentials(hyperscalerType Type) (Credentials, error) {
	labelSelector := fmt.Sprintf("shared=true,hyperscalerType=%s", hyperscalerType)
	secrets, err := getK8sSecrets(sp.secretsClient, labelSelector)
	if err != nil {
		return Credentials{}, err
	}

	secret, err := sp.getLeastUsed(secrets)
	if err != nil {
		return Credentials{}, err
	}

	return credentialsFromSecret(&secret, hyperscalerType), nil
}


func (sp *SharedAccountPool) getLeastUsed(secrets []apiv1.Secret) (apiv1.Secret, error) {
	usageCount := make(map[string]int, len(secrets))
	for _, s := range secrets {
		usageCount[s.Name] = 0
	}

	shoots, err := sp.shootsClient.List(metav1.ListOptions{})
	if err != nil {
		return apiv1.Secret{}, errors.Wrap(err, "sharedAccountPool error while listing Shoots")
	}

	if shoots == nil || len(shoots.Items) == 0 {
		return secrets[0], nil
	}

	for _, s := range shoots.Items {
		count, found := usageCount[s.Spec.SecretBindingName]
		if !found {
			continue
		}

		usageCount[s.Spec.SecretBindingName] = count + 1
	}

	min := usageCount[secrets[0].Name]
	minIndex := 0

	for i, s := range secrets {
		if usageCount[s.Name] < min {
			min = usageCount[s.Name]
			minIndex = i
		}
	}

	return secrets[minIndex], nil
}

func getK8sSecrets(secretsClient corev1.SecretInterface, labelSelector string) ([]apiv1.Secret, error) {
	secrets, err := secretsClient.List(metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil,
			errors.Wrapf(err, "sharedAccountPool error during secret list for LabelSelector: %s", labelSelector)
	}
	if secrets == nil || len(secrets.Items) == 0 {
		return nil, errors.Errorf("sharedAccountPool error: no shared Secret found for %s label selector", labelSelector)
	}

	return secrets.Items, nil
}
