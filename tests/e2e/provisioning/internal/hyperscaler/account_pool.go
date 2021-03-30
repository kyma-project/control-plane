package hyperscaler

import (
	"fmt"
	"sync"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_apis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
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
	CredentialsSecretBinding(hyperscalerType Type, tenantName string) (*v1beta1.SecretBinding, error)
}

type secretBindingsAccountPool struct {
	secretBindingsClient gardener_apis.SecretBindingInterface
	mux                  sync.Mutex
}

// NewAccountPool returns a new AccountPool
func NewAccountPool(secretBindingsClient gardener_apis.SecretBindingInterface) AccountPool {
	return &secretBindingsAccountPool{
		secretBindingsClient: secretBindingsClient,
	}
}

// Credentials returns the hyperscaler secret from k8s secret
func (p *secretBindingsAccountPool) CredentialsSecretBinding(hyperscalerType Type, tenantName string) (*v1beta1.SecretBinding, error) {
	labelSelector := fmt.Sprintf("tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)
	secretBinding, err := getSecretBinding(p.secretBindingsClient, labelSelector)
	if err != nil {
		return nil, err
	}
	if secretBinding != nil {
		return secretBinding, nil
	}

	// lock so that only one thread can fetch an unassigned secret binding and assign it (update secret binding with tenantName)
	p.mux.Lock()
	defer p.mux.Unlock()

	labelSelector = fmt.Sprintf("shared!=true, !tenantName, !dirty, hyperscalerType=%s", hyperscalerType)
	secretBinding, err = getSecretBinding(p.secretBindingsClient, labelSelector)
	if err != nil {
		return nil, err
	}

	if secretBinding == nil {
		return nil, errors.Errorf("failed to find unassigned secret binding for hyperscalerType: %s", hyperscalerType)
	}

	secretBinding.Labels["tenantName"] = tenantName
	updatedSecretBinding, err := p.secretBindingsClient.Update(secretBinding)
	if err != nil {
		return nil, errors.Wrapf(err, "updating secret binding with tenantName: %s", tenantName)
	}

	return updatedSecretBinding, nil
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
