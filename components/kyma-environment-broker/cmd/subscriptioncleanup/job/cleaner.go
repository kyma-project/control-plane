package job

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_apis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/subscriptioncleanup/cloudprovider"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/subscriptioncleanup/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Type string

type Cleaner interface {
	Do() error
}

func NewCleaner(context context.Context,
	kubernetesInterface kubernetes.Interface,
	secretBindingsClient gardener_apis.SecretBindingInterface,
	shootClient gardener_apis.ShootInterface,
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
	secretBindingsClient gardener_apis.SecretBindingInterface
	providerFactory      cloudprovider.ProviderFactory
	shootClient          gardener_apis.ShootInterface
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
			logrus.Errorf("Failed to list shoots for '%s' secret binding: %s", secretBinding.Name, err.Error())
			continue
		}

		if !canRelease {
			logrus.Warnf("Cannot release secret binding: %s. Still in use", secretBinding.Name)
			continue
		}

		err = p.releaseResources(secretBinding)
		if err != nil {
			logrus.Errorf("Failed to release resources for '%s' secret binding: %s", secretBinding.Name, err.Error())
			continue
		}
		err = p.returnSecretBindingToThePool(secretBinding)
		if err != nil {
			logrus.Errorf("Failed returning '%s' secret binding to the pool: %s", secretBinding.Name, err.Error())
			continue
		}
		logrus.Infof("Resources released for '%s' secret binding", secretBinding.Name)
	}

	logrus.Info("Finished releasing resources")
	return nil
}

func (p *cleaner) releaseResources(secretBinding v1beta1.SecretBinding) error {
	hyperscalerType, err := model.NewHyperscalerType(secretBinding.Labels["hyperscalerType"])
	if err != nil {
		return errors.Wrap(err, "starting releasing resources")
	}

	secret, err := p.getBoundSecret(secretBinding)
	if err != nil {
		return errors.Wrap(err, "getting referenced secret")
	}

	cleaner, err := p.providerFactory.New(hyperscalerType, secret.Data)
	if err != nil {
		return errors.Wrap(err, "initializing cloud provider cleaner")
	}

	return cleaner.Do()
}

func (p *cleaner) getBoundSecret(secretBinding v1beta1.SecretBinding) (*apiv1.Secret, error) {
	secret, err := p.kubernetesInterface.CoreV1().
		Secrets(secretBinding.SecretRef.Namespace).
		Get(p.context, secretBinding.SecretRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "getting %s/%s secret",
			secretBinding.SecretRef.Namespace, secretBinding.SecretRef.Name)
	}
	return secret, nil
}

func (p *cleaner) returnSecretBindingToThePool(secretBinding v1beta1.SecretBinding) error {
	sb, err := p.secretBindingsClient.Get(p.context, secretBinding.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	delete(sb.Labels, "dirty")
	delete(sb.Labels, "tenantName")

	_, err = p.secretBindingsClient.Update(p.context, sb, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to return secret binding to the hyperscaler account pool")
	}

	return nil
}

func (p *cleaner) getSecretBindingsToRelease() ([]v1beta1.SecretBinding, error) {
	labelSelector := fmt.Sprintf("dirty=true")

	return getSecretBindings(p.context, p.secretBindingsClient, labelSelector)
}

// Checks if there are no clusters tied to the secret binding
func (p *cleaner) checkIfSecretCanBeReleased(binding v1beta1.SecretBinding) (bool, error) {
	list, err := p.shootClient.List(p.context, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to list shoots")
	}

	for _, shoot := range list.Items {
		if shoot.Spec.SecretBindingName == binding.Name {
			return false, nil
		}
	}

	return true, nil
}

func getSecretBindings(ctx context.Context, secretBindingsClient gardener_apis.SecretBindingInterface, labelSelector string) ([]v1beta1.SecretBinding, error) {
	secrets, err := secretBindingsClient.List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil,
			errors.Wrapf(err, "listing secrets bindings for LabelSelector: %s", labelSelector)
	}

	return secrets.Items, nil
}
