package hyperscaler

import (
	"context"
	"fmt"
	"sync"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_apis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Type string

const (
	GCP       Type = "gcp"
	Azure     Type = "azure"
	AWS       Type = "aws"
	Openstack Type = "openstack"
)

type AccountPool interface {
	CredentialsSecretBinding(hyperscalerType Type, tenantName string) (*v1beta1.SecretBinding, error)
	MarkSecretBindingAsDirty(hyperscalerType Type, tenantName string) error
	IsSecretBindingUsed(hyperscalerType Type, tenantName string) (bool, error)
	IsSecretBindingDirty(hyperscalerType Type, tenantName string) (bool, error)
	IsSecretBindingInternal(hyperscalerType Type, tenantName string) (bool, error)
}

func NewAccountPool(secretBindingsClient gardener_apis.SecretBindingInterface, shootsClient gardener_apis.ShootInterface) AccountPool {
	return &secretBindingsAccountPool{
		secretBindingsClient: secretBindingsClient,
		shootsClient:         shootsClient,
	}
}

type secretBindingsAccountPool struct {
	secretBindingsClient gardener_apis.SecretBindingInterface
	shootsClient         gardener_apis.ShootInterface
	mux                  sync.Mutex
}

func (p *secretBindingsAccountPool) IsSecretBindingInternal(hyperscalerType Type, tenantName string) (bool, error) {
	labelSelector := fmt.Sprintf("internal=true, tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)
	secretBinding, err := p.getSecretBinding(labelSelector)
	if err != nil {
		return false, errors.Wrapf(err, "looking for a secret binding used by the tenant %s and hyperscaler %s", tenantName, hyperscalerType)
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
		return false, errors.Wrapf(err, "looking for a secret binding used by the tenant %s and hyperscaler %s", tenantName, hyperscalerType)
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
		return errors.Wrapf(err, "marking secret binding as dirty: failed to find secret binding used by the tenant %s and hyperscaler %s", tenantName, hyperscalerType)
	}

	secretBinding.Labels["dirty"] = "true"

	_, err = p.secretBindingsClient.Update(context.Background(), secretBinding, v1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "marking secret binding as dirty: failed to update secret binding for tenant: %s and hyperscaler: %s", tenantName, hyperscalerType)
	}
	return nil
}

func (p *secretBindingsAccountPool) IsSecretBindingUsed(hyperscalerType Type, tenantName string) (bool, error) {
	labelSelector := fmt.Sprintf("tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)
	secretBinding, err := p.getSecretBinding(labelSelector)
	if err != nil || secretBinding == nil {
		return false, errors.Wrapf(err, "counting subscription usage: could not find secret binding used by the tenant %s and hyperscaler %s", tenantName, hyperscalerType)
	}

	shootlist, err := p.shootsClient.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "listing Gardener shoots")
	}

	for _, shoot := range shootlist.Items {
		if shoot.Spec.SecretBindingName == secretBinding.Name {
			return true, nil
		}
	}

	return false, nil
}

func (p *secretBindingsAccountPool) CredentialsSecretBinding(hyperscalerType Type, tenantName string) (*v1beta1.SecretBinding, error) {
	labelSelector := fmt.Sprintf("tenantName=%s, hyperscalerType=%s, !dirty", tenantName, hyperscalerType)
	secretBinding, err := p.getSecretBinding(labelSelector)
	if err != nil {
		return nil, errors.Wrap(err, "getting secret binding")
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
		return nil, errors.Wrap(err, "getting secret binding")
	}
	if secretBinding == nil {
		return nil, errors.Errorf("failed to find unassigned secret binding for hyperscalerType: %s",
			hyperscalerType)
	}

	secretBinding.Labels["tenantName"] = tenantName
	updatedSecretBinding, err := p.secretBindingsClient.Update(context.Background(), secretBinding, v1.UpdateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "updating secret binding with tenantName: %s", tenantName)
	}

	return updatedSecretBinding, nil
}

func (p *secretBindingsAccountPool) getSecretBinding(labelSelector string) (*v1beta1.SecretBinding, error) {
	secretBindings, err := p.secretBindingsClient.List(context.Background(), metav1.ListOptions{
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
