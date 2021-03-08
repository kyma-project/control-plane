package hyperscaler

import (
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_apis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"k8s.io/client-go/kubernetes"
	"sync"

	"github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Type string

const (
	Azure Type = "azure"
)

type Credentials struct {
	Name            string
	TenantName      string
	HyperscalerType Type
	CredentialData  map[string][]byte
}

type AccountPool interface {
	Credentials(hyperscalerType Type, tenantName string) (Credentials, error)
}

type secretBindingsAccountPool struct {
	kubernetesInterface  kubernetes.Interface
	secretBindingsClient gardener_apis.SecretBindingInterface
	mux                  sync.Mutex
}

// NewAccountPool returns a new AccountPool
func NewAccountPool(kubernetesInterface kubernetes.Interface, secretBindingsClient gardener_apis.SecretBindingInterface) AccountPool {
	return &secretBindingsAccountPool{
		kubernetesInterface:  kubernetesInterface,
		secretBindingsClient: secretBindingsClient,
	}
}

// Credentials returns the hyperscaler secret from k8s secret
func (p *secretBindingsAccountPool) Credentials(hyperscalerType Type, tenantName string) (Credentials, error) {
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
		return Credentials{}, errors.Wrapf(err, "updating secret binding with tenantName: %s", tenantName)
	}

	return credentialsFromBoundSecret(p.kubernetesInterface, updatedSecretBinding, hyperscalerType)
}

func getSecretBinding(secretBindingsClient gardener_apis.SecretBindingInterface, labelSelector string) (*v1beta1.SecretBinding, error) {
	secretBindings, err := secretBindingsClient.List(metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "listing secret bindings for LabelSelector: %s", labelSelector)
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
		TenantName:      tenantName,
		HyperscalerType: hyperscalerType,
		CredentialData:  secret.Data,
	}, nil
}
