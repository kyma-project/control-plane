package job

import (
	"context"
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/subscriptioncleanup/cloudprovider"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/subscriptioncleanup/model"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type Type string

type Cleaner interface {
	Do() error
}

func NewCleaner(context context.Context,
	kubernetesInterface kubernetes.Interface,
	secretBindingsClient dynamic.ResourceInterface,
	shootClient dynamic.ResourceInterface,
	providerFactory cloudprovider.ProviderFactory) Cleaner {

	return &cleaner{
		kubernetesInterface:  kubernetesInterface,
		secretBindingsClient: secretBindingsClient,
		providerFactory:      providerFactory,
		shootClient:          shootClient,
		context:              context,
	}
}

type cleaner struct {
	kubernetesInterface  kubernetes.Interface
	secretBindingsClient dynamic.ResourceInterface
	providerFactory      cloudprovider.ProviderFactory
	shootClient          dynamic.ResourceInterface
	context              context.Context
}

func (p *cleaner) Do() error {
	logrus.Info("Started releasing resources")
	secretBindings, err := p.getSecretBindingsToRelease()
	if err != nil {
		return err
	}
	for _, secretBinding := range secretBindings {
		canRelease, err := p.checkIfSecretCanBeReleased(secretBinding)
		if err != nil {
			logrus.Errorf("Failed to list shoots for '%s' secret binding: %s", secretBinding.GetName(), err.Error())
			continue
		}

		if !canRelease {
			logrus.Warnf("Cannot release secret binding: %s. Still in use", secretBinding.GetName())
			continue
		}

		err = p.releaseResources(secretBinding)
		if err != nil {
			logrus.Errorf("Failed to release resources for '%s' secret binding: %s", secretBinding.GetName(), err.Error())
			continue
		}
		err = p.returnSecretBindingToThePool(secretBinding)
		if err != nil {
			logrus.Errorf("Failed returning '%s' secret binding to the pool: %s", secretBinding.GetName(), err.Error())
			continue
		}
		logrus.Infof("Resources released for '%s' secret binding", secretBinding.GetName())
	}

	logrus.Info("Finished releasing resources")
	return nil
}

func (p *cleaner) releaseResources(secretBinding unstructured.Unstructured) error {
	hyperscalerType, err := model.NewHyperscalerType(secretBinding.GetLabels()["hyperscalerType"])
	if err != nil {
		return fmt.Errorf("starting releasing resources: %w", err)
	}

	secret, err := p.getBoundSecret(secretBinding)
	if err != nil {
		return fmt.Errorf("getting referenced secret: %w", err)
	}

	cleaner, err := p.providerFactory.New(hyperscalerType, secret.Data)
	if err != nil {
		return fmt.Errorf("initializing cloud provider cleaner: %w", err)
	}

	return cleaner.Do()
}

func (p *cleaner) getBoundSecret(sb unstructured.Unstructured) (*apiv1.Secret, error) {
	secretBinding := gardener.SecretBinding{sb}
	secret, err := p.kubernetesInterface.CoreV1().
		Secrets(secretBinding.GetSecretRefNamespace()).
		Get(p.context, secretBinding.GetSecretRefName(), metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting %s/%s secret: %w", secretBinding.GetSecretRefNamespace(),
			secretBinding.GetSecretRefName(), err)
	}
	return secret, nil
}

func (p *cleaner) returnSecretBindingToThePool(secretBinding unstructured.Unstructured) error {
	sb, err := p.secretBindingsClient.Get(p.context, secretBinding.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}

	l := sb.GetLabels()
	delete(l, "dirty")
	delete(l, "tenantName")
	sb.SetLabels(l)

	_, err = p.secretBindingsClient.Update(p.context, sb, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to return secret binding to the hyperscaler account pool: %w", err)
	}

	return nil
}

func (p *cleaner) getSecretBindingsToRelease() ([]unstructured.Unstructured, error) {
	labelSelector := fmt.Sprintf("dirty=true")

	return getSecretBindings(p.context, p.secretBindingsClient, labelSelector)
}

// Checks if there are no clusters tied to the secret binding
func (p *cleaner) checkIfSecretCanBeReleased(binding unstructured.Unstructured) (bool, error) {
	list, err := p.shootClient.List(p.context, metav1.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to list shoots: %w", err)
	}

	for _, sh := range list.Items {
		shoot := gardener.Shoot{sh}
		if shoot.GetSpecSecretBindingName() == binding.GetName() {
			return false, nil
		}
	}

	return true, nil
}

func getSecretBindings(ctx context.Context, secretBindingsClient dynamic.ResourceInterface, labelSelector string) ([]unstructured.Unstructured, error) {
	secrets, err := secretBindingsClient.List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("listing secrets bindings for LabelSelector: %s: %w", labelSelector, err)
	}

	return secrets.Items, nil
}
