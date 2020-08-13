package hyperscaler

import (
	"fmt"
	"strings"
	"sync"

	gardener_apis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/pkg/errors"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type Type string

const (
	GCP   Type = "gcp"
	Azure Type = "azure"
	AWS   Type = "aws"
)

func HyperscalerTypeFromProviderString(provider string) (Type, error) {

	hyperscalerType := Type(strings.ToLower(provider))

	switch hyperscalerType {
	case GCP, Azure, AWS:
		return hyperscalerType, nil
	}
	return "", errors.Errorf("unknown Hyperscaler provider type: %s", provider)
}

type Credentials struct {
	Name            string
	HyperscalerType Type
	CredentialData  map[string][]byte
}

type AccountPool interface {
	Credentials(hyperscalerType Type, tenantName string) (Credentials, error)
	MarkSecretAsDirty(hyperscalerType Type, tenantName string) error
	// Consider moving this method to Account Provider
	IsSecretUsed(hyperscalerType Type, tenantName string) (bool, error)
	IsSecretDirty(hyperscalerType Type, tenantName string) (bool, error)
}

func NewAccountPool(secretsClient corev1.SecretInterface, shootsClient gardener_apis.ShootInterface) AccountPool {
	return &secretsAccountPool{
		secretsClient: secretsClient,
		shootsClient:  shootsClient,
	}
}

type secretsAccountPool struct {
	secretsClient corev1.SecretInterface
	shootsClient  gardener_apis.ShootInterface
	mux           sync.Mutex
}

func (p *secretsAccountPool) IsSecretDirty(hyperscalerType Type, tenantName string) (bool, error) {

	labelSelector := fmt.Sprintf("shared!=true, dirty=true, tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)

	secret, err := getK8SSecret(p.secretsClient, labelSelector)

	if err != nil {
		return false, errors.Wrapf(err, "error while looking for a secret used by the tenant %s and hyperscaler %s", tenantName, hyperscalerType)
	}

	if secret != nil {
		return true, nil
	}

	return false, nil
}

func (p *secretsAccountPool) MarkSecretAsDirty(hyperscalerType Type, tenantName string) error {

	p.mux.Lock()
	defer p.mux.Unlock()

	labelSelector := fmt.Sprintf("shared!=true, tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)

	secret, err := getK8SSecret(p.secretsClient, labelSelector)

	if err != nil || secret == nil {
		return errors.Wrapf(err, "accountPool failed to find secret used by the tenant %s and hyperscaler %s to release subscription", tenantName, hyperscalerType)
	}

	secret.Labels["dirty"] = "true"

	_, err = p.secretsClient.Update(secret)
	if err != nil {
		return errors.Wrapf(err, "accountPool failed to update secret with dirty label for tenant: %s and hyperscaler: %s", tenantName, hyperscalerType)
	}

	return nil
}

func (p *secretsAccountPool) IsSecretUsed(hyperscalerType Type, tenantName string) (bool, error) {

	labelSelector := fmt.Sprintf("tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)

	secret, err := getK8SSecret(p.secretsClient, labelSelector)

	if err != nil || secret == nil {
		return false, errors.Wrapf(err, "Could not find secret used by the tenant %s and hyperscaler %s to count subscription usage", tenantName, hyperscalerType)
	}

	shootlist, err := p.shootsClient.List(metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "Error while listing Gardener shoots ")
	}

	for _, shoot := range shootlist.Items {
		if shoot.Spec.SecretBindingName == secret.Name {
			return true, nil
		}
	}

	return false, nil
}

func (p *secretsAccountPool) Credentials(hyperscalerType Type, tenantName string) (Credentials, error) {

	labelSelector := fmt.Sprintf("tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)
	secret, err := getK8SSecret(p.secretsClient, labelSelector)

	if err != nil {
		return Credentials{}, err
	}
	if secret != nil {
		return credentialsFromSecret(secret, hyperscalerType), nil
	}

	labelSelector = fmt.Sprintf("shared!=true, !tenantName, !dirty, hyperscalerType=%s", hyperscalerType)
	// lock so that only one thread can fetch an unassigned secret and assign it (update secret with tenantName)
	p.mux.Lock()
	defer p.mux.Unlock()
	secret, err = getK8SSecret(p.secretsClient, labelSelector)

	if err != nil {
		return Credentials{}, err
	}

	if secret == nil {
		return Credentials{}, errors.Errorf("accountPool failed to find unassigned secret for hyperscalerType: %s", hyperscalerType)
	}

	secret.Labels["tenantName"] = tenantName
	updatedSecret, err := p.secretsClient.Update(secret)
	if err != nil {
		return Credentials{}, errors.Wrapf(err, "accountPool error while updating secret with tenantName: %s", tenantName)
	}

	return credentialsFromSecret(updatedSecret, hyperscalerType), nil
}

func getK8SSecret(secretsClient corev1.SecretInterface, labelSelector string) (*apiv1.Secret, error) {
	secrets, err := secretsClient.List(metav1.ListOptions{
		LabelSelector: labelSelector,
	})

	if err != nil {
		return nil,
			errors.Wrapf(err, "accountPool error during secret list for LabelSelector: %s", labelSelector)
	}

	if secrets != nil && len(secrets.Items) > 0 {
		return &secrets.Items[0], nil
	}

	return nil, nil
}

func credentialsFromSecret(secret *apiv1.Secret, hyperscalerType Type) Credentials {
	return Credentials{
		Name:            secret.Name,
		HyperscalerType: hyperscalerType,
		CredentialData:  secret.Data,
	}
}
