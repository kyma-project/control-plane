package hyperscaler

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"sync"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_apis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Type string

const (
	GCP   Type = "gcp"
	Azure Type = "azure"
	AWS   Type = "aws"
)

type Credentials struct {
	Name            string
	HyperscalerType Type
	CredentialData  map[string][]byte
}

type AccountPool interface {
	Credentials(hyperscalerType Type, tenantName string) (Credentials, error)
	MarkSecretBindingAsDirty(hyperscalerType Type, tenantName string) error
	IsSecretBindingUsed(hyperscalerType Type, tenantName string) (bool, error)
	IsSecretBindingDirty(hyperscalerType Type, tenantName string) (bool, error)
	IsSecretBindingInternal(hyperscalerType Type, tenantName string) (bool, error)
}

func NewAccountPool(kubernetesInterface kubernetes.Interface, secretBindingsClient gardener_apis.SecretBindingInterface, shootsClient gardener_apis.ShootInterface) AccountPool {
	return &secretsAccountPool{
		kubernetesInterface:  kubernetesInterface,
		secretBindingsClient: secretBindingsClient,
		shootsClient:         shootsClient,
	}
}

type secretsAccountPool struct {
	kubernetesInterface  kubernetes.Interface
	secretBindingsClient gardener_apis.SecretBindingInterface
	shootsClient         gardener_apis.ShootInterface
	mux                  sync.Mutex
}

func (p *secretsAccountPool) IsSecretBindingInternal(hyperscalerType Type, tenantName string) (bool, error) {
	labelSelector := fmt.Sprintf("internal=true, tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)
	secretBinding, err := getSecretBinding(p.secretBindingsClient, labelSelector)
	if err != nil {
		return false, errors.Wrapf(err, "error looking for a secret binding used by the tenant %s and hyperscaler %s", tenantName, hyperscalerType)
	}

	if secretBinding != nil {
		return true, nil
	}
	return false, nil
}

func (p *secretsAccountPool) IsSecretBindingDirty(hyperscalerType Type, tenantName string) (bool, error) {
	labelSelector := fmt.Sprintf("shared!=true, dirty=true, tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)
	secretBinding, err := getSecretBinding(p.secretBindingsClient, labelSelector)
	if err != nil {
		return false, errors.Wrapf(err, "error looking for a secret binding used by the tenant %s and hyperscaler %s", tenantName, hyperscalerType)
	}

	if secretBinding != nil {
		return true, nil
	}
	return false, nil
}

func (p *secretsAccountPool) MarkSecretBindingAsDirty(hyperscalerType Type, tenantName string) error {
	p.mux.Lock()
	defer p.mux.Unlock()

	labelSelector := fmt.Sprintf("shared!=true, tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)
	secretBinding, err := getSecretBinding(p.secretBindingsClient, labelSelector)
	if err != nil || secretBinding == nil {
		return errors.Wrapf(err, "error marking secret binding as dirty: failed to find secret binding used by the tenant %s and hyperscaler %s", tenantName, hyperscalerType)
	}

	secretBinding.Labels["dirty"] = "true"

	_, err = p.secretBindingsClient.Update(secretBinding)
	if err != nil {
		return errors.Wrapf(err, "error marking secret binding as dirty: failed to update secret binding for tenant: %s and hyperscaler: %s", tenantName, hyperscalerType)
	}
	return nil
}

func (p *secretsAccountPool) IsSecretBindingUsed(hyperscalerType Type, tenantName string) (bool, error) {
	labelSelector := fmt.Sprintf("tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)
	secretBinding, err := getSecretBinding(p.secretBindingsClient, labelSelector)
	if err != nil || secretBinding == nil {
		return false, errors.Wrapf(err, "error counting subscription usage: could not find secret binding used by the tenant %s and hyperscaler %s", tenantName, hyperscalerType)
	}

	shootlist, err := p.shootsClient.List(metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "error while listing Gardener shoots")
	}

	for _, shoot := range shootlist.Items {
		if shoot.Spec.SecretBindingName == secretBinding.Name {
			return true, nil
		}
	}

	return false, nil
}

func (p *secretsAccountPool) Credentials(hyperscalerType Type, tenantName string) (Credentials, error) {
	labelSelector := fmt.Sprintf("tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)
	secretBinding, err := getSecretBinding(p.secretBindingsClient, labelSelector)
	if err != nil {
		return Credentials{}, err
	}
	if secretBinding != nil {
		return credentialsFromBoundSecret(p.kubernetesInterface, secretBinding, hyperscalerType)
	}

	// lock so that only one thread can fetch an unassigned secret and assign it (update secret with tenantName)
	p.mux.Lock()
	defer p.mux.Unlock()

	labelSelector = fmt.Sprintf("shared!=true, !tenantName, !dirty, hyperscalerType=%s", hyperscalerType)
	secretBinding, err = getSecretBinding(p.secretBindingsClient, labelSelector)
	if err != nil {
		return Credentials{}, err
	}

	if secretBinding == nil {
		return Credentials{}, errors.Errorf("failed to find unassigned secret binding for hyperscalerType: %s", hyperscalerType)
	}

	secretBinding.Labels["tenantName"] = tenantName
	updatedSecretBinding, err := p.secretBindingsClient.Update(secretBinding)
	if err != nil {
		return Credentials{}, errors.Wrapf(err, "error updating secret binding with tenantName: %s", tenantName)
	}

	return credentialsFromBoundSecret(p.kubernetesInterface, updatedSecretBinding, hyperscalerType)
}

func getSecretBinding(secretBindingsClient gardener_apis.SecretBindingInterface, labelSelector string) (*v1beta1.SecretBinding, error) {
	secretBindings, err := secretBindingsClient.List(metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "error listing secret bindings for LabelSelector: %s", labelSelector)
	}

	if secretBindings != nil && len(secretBindings.Items) > 0 {
		return &secretBindings.Items[0], nil
	}
	return nil, nil
}

func credentialsFromBoundSecret(kubernetesInterface kubernetes.Interface, secretBinding *v1beta1.SecretBinding, hyperscalerType Type) (Credentials, error) {
	secretClient := kubernetesInterface.CoreV1().Secrets(secretBinding.SecretRef.Namespace)

	secret, err := secretClient.Get(secretBinding.SecretRef.Name, metav1.GetOptions{})
	if err != nil {
		return Credentials{}, errors.Wrapf(err, "getting %s/%s secret", secretBinding.SecretRef.Namespace, secretBinding.SecretRef.Name)
	}

	return Credentials{
		Name:            secret.Name,
		HyperscalerType: hyperscalerType,
		CredentialData:  secret.Data,
	}, nil
}
