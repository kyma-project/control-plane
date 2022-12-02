package hyperscaler

import (
	"context"
	"fmt"
	"sync"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

type Type string

const (
	GCP       Type = "gcp"
	Azure     Type = "azure"
	AWS       Type = "aws"
	Openstack Type = "openstack"
)

type AccountPool interface {
	CredentialsSecretBinding(hyperscalerType Type, tenantName string) (*gardener.SecretBinding, error)
	MarkSecretBindingAsDirty(hyperscalerType Type, tenantName string) error
	IsSecretBindingUsed(hyperscalerType Type, tenantName string) (bool, error)
	IsSecretBindingDirty(hyperscalerType Type, tenantName string) (bool, error)
	IsSecretBindingInternal(hyperscalerType Type, tenantName string) (bool, error)
}

func NewAccountPool(gardenerClient dynamic.Interface, gardenerNamespace string) AccountPool {
	return &secretBindingsAccountPool{
		gardenerClient: gardenerClient,
		gardenerNS:     gardenerNamespace,
	}
}

type secretBindingsAccountPool struct {
	gardenerClient dynamic.Interface
	gardenerNS     string
	mux            sync.Mutex
}

func (p *secretBindingsAccountPool) IsSecretBindingInternal(hyperscalerType Type, tenantName string) (bool, error) {
	labelSelector := fmt.Sprintf("internal=true, tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)
	secretBinding, err := p.getSecretBinding(labelSelector)
	if err != nil {
		return false, fmt.Errorf("looking for a secret binding used by the tenant %s and hyperscaler %s: %w", tenantName, hyperscalerType, err)
	}

	if secretBinding != nil {
		return true, nil
	}
	return false, nil
}

func (p *secretBindingsAccountPool) IsSecretBindingDirty(hyperscalerType Type, tenantName string) (bool, error) {
	labelSelector := fmt.Sprintf("shared!=true, dirty=true, tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)
	secretBinding, err := p.getSecretBinding(labelSelector)
	if err != nil {
		return false, fmt.Errorf("looking for a secret binding used by the tenant %s and hyperscaler %s: %w", tenantName, hyperscalerType, err)
	}

	if secretBinding != nil {
		return true, nil
	}
	return false, nil
}

func (p *secretBindingsAccountPool) MarkSecretBindingAsDirty(hyperscalerType Type, tenantName string) error {
	p.mux.Lock()
	defer p.mux.Unlock()

	labelSelector := fmt.Sprintf("shared!=true, tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)
	secretBinding, err := p.getSecretBinding(labelSelector)
	if err != nil || secretBinding == nil {
		return fmt.Errorf("marking secret binding as dirty: failed to find secret binding used by the tenant %s and"+" hyperscaler %s: %w", tenantName, hyperscalerType, err)
	}

	labels := secretBinding.GetLabels()
	labels["dirty"] = "true"
	secretBinding.SetLabels(labels)

	_, err = p.gardenerClient.Resource(gardener.SecretBindingResource).Namespace(p.gardenerNS).Update(context.Background(), &secretBinding.Unstructured, v1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("marking secret binding as dirty: failed to update secret binding for tenant: %s and hyperscaler: %s: %w", tenantName, hyperscalerType, err)

	}
	return nil
}

func (p *secretBindingsAccountPool) IsSecretBindingUsed(hyperscalerType Type, tenantName string) (bool, error) {
	labelSelector := fmt.Sprintf("tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)
	secretBinding, err := p.getSecretBinding(labelSelector)
	if err != nil || secretBinding == nil {
		return false, fmt.Errorf("counting subscription usage: could not find secret binding used by the tenant %s and hyperscaler %s: %w", tenantName, hyperscalerType, err)
	}

	shootlist, err := p.gardenerClient.Resource(gardener.ShootResource).Namespace(p.gardenerNS).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("listing Gardener shoots: %w", err)
	}

	for _, shoot := range shootlist.Items {
		sh := gardener.Shoot{shoot}
		if sh.GetSpecSecretBindingName() == secretBinding.GetName() {
			return true, nil
		}
	}

	return false, nil
}

func (p *secretBindingsAccountPool) CredentialsSecretBinding(hyperscalerType Type, tenantName string) (*gardener.SecretBinding, error) {
	labelSelector := fmt.Sprintf("tenantName=%s, hyperscalerType=%s, !dirty", tenantName, hyperscalerType)
	secretBinding, err := p.getSecretBinding(labelSelector)
	if err != nil {
		return nil, fmt.Errorf("getting secret binding: %w", err)
	}
	if secretBinding != nil {
		return secretBinding, nil
	}

	// lock so that only one thread can fetch an unassigned secret binding and assign it
	// (update secret binding with tenantName)
	p.mux.Lock()
	defer p.mux.Unlock()

	labelSelector = fmt.Sprintf("shared!=true, !tenantName, !dirty, hyperscalerType=%s", hyperscalerType)
	secretBinding, err = p.getSecretBinding(labelSelector)
	if err != nil {
		return nil, fmt.Errorf("getting secret binding: %w", err)
	}
	if secretBinding == nil {
		return nil, fmt.Errorf("failed to find unassigned secret binding for hyperscalerType: %s", hyperscalerType)
	}

	labels := secretBinding.GetLabels()
	labels["tenantName"] = tenantName
	secretBinding.SetLabels(labels)
	updatedSecretBinding, err := p.gardenerClient.Resource(gardener.SecretBindingResource).Namespace(p.gardenerNS).Update(context.Background(), &secretBinding.Unstructured, v1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating secret binding with tenantName: %s: %w", tenantName, err)
	}

	return &gardener.SecretBinding{*updatedSecretBinding}, nil
}

func (p *secretBindingsAccountPool) getSecretBinding(labelSelector string) (*gardener.SecretBinding, error) {
	secretBindings, err := p.gardenerClient.Resource(gardener.SecretBindingResource).Namespace(p.gardenerNS).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("listing secret bindings for LabelSelector: %s: %w", labelSelector, err)
	}

	if secretBindings != nil && len(secretBindings.Items) > 0 {
		return &gardener.SecretBinding{secretBindings.Items[0]}, nil
	}
	return nil, nil
}
