package hyperscaler

import (
	"fmt"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type AccountProvider interface {
	GardenerCredentials(hyperscalerType Type, tenantName string) (Credentials, error)
}

type accountProvider struct {
	kubernetesInterface kubernetes.Interface
	compassPool         AccountPool
	gardenerPool        AccountPool
}

// NewAccountProvider returns a new AccountProvider
func NewAccountProvider(kubernetesInterface kubernetes.Interface, compassPool AccountPool, gardenerPool AccountPool) AccountProvider {
	return &accountProvider{
		kubernetesInterface: kubernetesInterface,
		compassPool:         compassPool,
		gardenerPool:        gardenerPool,
	}
}

// GardenerCredentials returns credentials for Gardener account
func (p *accountProvider) GardenerCredentials(hyperscalerType Type, tenantName string) (Credentials, error) {
	if p.gardenerPool == nil {
		return Credentials{},
			fmt.Errorf("failed to get Gardener Credentials. Gardener Account pool is not configured for tenant: %s", tenantName)
	}

	secretBinding, err := p.gardenerPool.CredentialsSecretBinding(hyperscalerType, tenantName)
	if err != nil {
		return Credentials{}, err
	}

	return p.credentialsFromBoundSecret(secretBinding, hyperscalerType, tenantName)
}

func (p *accountProvider) credentialsFromBoundSecret(secretBinding *v1beta1.SecretBinding, hyperscalerType Type, tenantName string) (Credentials, error) {
	secretClient := p.kubernetesInterface.CoreV1().Secrets(secretBinding.SecretRef.Namespace)

	secret, err := secretClient.Get(secretBinding.SecretRef.Name, metav1.GetOptions{})
	if err != nil {
		return Credentials{}, errors.Wrapf(err, "getting %s/%s secret",
			secretBinding.SecretRef.Namespace, secretBinding.SecretRef.Name)
	}

	return Credentials{
		Name:            secret.Name,
		TenantName:      tenantName,
		HyperscalerType: hyperscalerType,
		CredentialData:  secret.Data,
	}, nil
}
