package hyperscaler

import (
	"context"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//go:generate mockery -name=AccountProvider -output=automock -outpkg=automock -case=underscore
type AccountProvider interface {
	GardenerCredentials(hyperscalerType Type, tenantName string) (Credentials, error)
	GardenerSharedCredentials(hyperscalerType Type) (Credentials, error)
	MarkUnusedGardenerSecretBindingAsDirty(hyperscalerType Type, tenantName string) error
}

type Credentials struct {
	Name            string
	HyperscalerType Type
	CredentialData  map[string][]byte
}

type accountProvider struct {
	kubernetesInterface kubernetes.Interface
	gardenerPool        AccountPool
	sharedGardenerPool  SharedPool
}

func NewAccountProvider(kubernetesInterface kubernetes.Interface, gardenerPool AccountPool, sharedGardenerPool SharedPool) AccountProvider {
	return &accountProvider{
		kubernetesInterface: kubernetesInterface,
		gardenerPool:        gardenerPool,
		sharedGardenerPool:  sharedGardenerPool,
	}
}

func FromCloudProvider(cp internal.CloudProvider) (Type, error) {
	switch cp {
	case internal.Azure:
		return Azure, nil
	case internal.AWS:
		return AWS, nil
	case internal.GCP:
		return GCP, nil
	case internal.Openstack:
		return Openstack, nil
	default:
		return "", errors.Errorf("cannot determine the type of Hyperscaler to use for cloud provider %s", cp)
	}
}

func (p *accountProvider) GardenerCredentials(hyperscalerType Type, tenantName string) (Credentials, error) {
	if p.gardenerPool == nil {
		return Credentials{},
			errors.New("failed to get Gardener Credentials. Gardener Account pool is not configured")
	}

	secretBinding, err := p.gardenerPool.CredentialsSecretBinding(hyperscalerType, tenantName)
	if err != nil {
		return Credentials{}, errors.Wrap(err, "getting credentials secret binding")
	}

	return p.credentialsFromBoundSecret(secretBinding, hyperscalerType)
}

func (p *accountProvider) GardenerSharedCredentials(hyperscalerType Type) (Credentials, error) {
	if p.sharedGardenerPool == nil {
		return Credentials{},
			errors.New("failed to get shared Gardener Credentials. Gardener Shared Account pool is not configured")
	}

	secretBinding, err := p.sharedGardenerPool.SharedCredentialsSecretBinding(hyperscalerType)
	if err != nil {
		return Credentials{}, errors.Wrap(err, "getting shared credentials secret binding")
	}

	return p.credentialsFromBoundSecret(secretBinding, hyperscalerType)
}

func (p *accountProvider) MarkUnusedGardenerSecretBindingAsDirty(hyperscalerType Type, tenantName string) error {
	if p.gardenerPool == nil {
		return errors.New("failed to release subscription for tenant. Gardener Account pool is not configured")
	}

	internal, err := p.gardenerPool.IsSecretBindingInternal(hyperscalerType, tenantName)
	if err != nil {
		return errors.Wrap(err, "checking if secret binding is internal")
	}
	if internal {
		return nil
	}

	dirty, err := p.gardenerPool.IsSecretBindingDirty(hyperscalerType, tenantName)
	if err != nil {
		return errors.Wrap(err, "checking if secret binding is dirty")
	}
	if dirty {
		return nil
	}

	secretBindingUsed, err := p.gardenerPool.IsSecretBindingUsed(hyperscalerType, tenantName)
	if err != nil {
		return errors.Wrapf(err, "cannot determine whether %s secret binding is used for tenant: %s", hyperscalerType, tenantName)
	}
	if !secretBindingUsed {
		return p.gardenerPool.MarkSecretBindingAsDirty(hyperscalerType, tenantName)
	}

	return nil
}

func (p *accountProvider) credentialsFromBoundSecret(secretBinding *v1beta1.SecretBinding, hyperscalerType Type) (Credentials, error) {
	secretClient := p.kubernetesInterface.CoreV1().Secrets(secretBinding.SecretRef.Namespace)

	secret, err := secretClient.Get(context.Background(), secretBinding.SecretRef.Name, metav1.GetOptions{})
	if err != nil {
		return Credentials{}, errors.Wrapf(err, "getting %s/%s secret", secretBinding.SecretRef.Namespace, secretBinding.SecretRef.Name)
	}

	return Credentials{
		Name:            secret.Name,
		HyperscalerType: hyperscalerType,
		CredentialData:  secret.Data,
	}, nil
}
