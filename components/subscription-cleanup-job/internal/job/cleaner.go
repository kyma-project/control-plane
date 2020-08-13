package job

import (
	"fmt"
	"github.com/kyma-project/control-plane/components/subscription-cleanup-job/internal/cloudprovider"
	"github.com/kyma-project/control-plane/components/subscription-cleanup-job/internal/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type Type string

type Cleaner interface {
	Do() error
}

func NewCleaner(secretsClient corev1.SecretInterface) Cleaner {
	return &cleaner{
		secretsClient: secretsClient,
	}
}

type cleaner struct {
	secretsClient corev1.SecretInterface
}

func (p *cleaner) Do() error {

	logrus.Info("Started releasing resources")
	secrets, err := p.getSecretsToRelease()
	if err != nil {
		return err
	}

	for _, secret := range secrets {
		err := p.releaseResources(secret)
		if err == nil {
			p.returnSecretToThePool(secret)
			logrus.Infof("Resources released for '%s' secret", secret.Name)
		}
	}

	return nil
}

func (p *cleaner) releaseResources(secret apiv1.Secret) error {
	hyperscalerType, err := model.NewHyperscalerType(secret.Labels["hyperscalerType"])
	if err != nil {
		return errors.Wrap(err, "failed to start releasing resources")
	}

	cleaner, err := cloudprovider.New(hyperscalerType, secret.Data)
	if err != nil {
		return errors.Wrap(err, "failed to initialize cloud provider cleaner")
	}

	return cleaner.Do()
}

func (p *cleaner) returnSecretToThePool(secret apiv1.Secret) error {

	s, err := p.secretsClient.Get(secret.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	delete(s.Labels, "dirty")
	delete(s.Labels, "tenantName")

	_, err = p.secretsClient.Update(s)
	if err != nil {
		return errors.Wrap(err, "failed to return secret to the hyperscaler account pool")
	}

	return nil
}

func (p *cleaner) getSecretsToRelease() ([]apiv1.Secret, error) {
	labelSelector := fmt.Sprintf("dirty=true")

	return getK8SSecrets(p.secretsClient, labelSelector)
}

func getK8SSecrets(secretsClient corev1.SecretInterface, labelSelector string) ([]apiv1.Secret, error) {
	secrets, err := secretsClient.List(metav1.ListOptions{
		LabelSelector: labelSelector,
	})

	if err != nil {
		return nil,
			errors.Wrapf(err, "failed to list secrets for LabelSelector: %s", labelSelector)
	}

	return secrets.Items, nil
}
