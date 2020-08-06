package hyperscaler

import (
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_apis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/pkg/errors"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"strings"
	"sync"
)

type Type string

const (
	GCP                    Type = "gcp"
	Azure                  Type = "azure"
	AWS                    Type = "aws"
	fieldSecretBindingName      = "spec.secretBindingName"
)

func HyperscalerTypeFromProviderString(provider string) (Type, error) {

	hyperscalerType := Type(strings.ToLower(provider))

	switch hyperscalerType {
	case GCP, Azure, AWS:
		return hyperscalerType, nil
	}
	return "", errors.Errorf("unknown Hyperscaler provider type: %s", provider)
}

type Credentials struct {
	Name            string
	HyperscalerType Type
	CredentialData  map[string][]byte
}

type AccountPool interface {
	Credentials(hyperscalerType Type, tenantName string) (Credentials, error)
	ReleaseSubscription(hyperscalerType Type, tenantName string) error
	CountSubscriptionUsages(hyperscalerType Type, tenantName string) (int, error)
	IsSubscriptionAlreadyReleased(hyperscalerType Type, tenantName string) (bool, error)
}

func NewAccountPool(secretsClient corev1.SecretInterface, shootsClient gardener_apis.ShootInterface) AccountPool {
	return &secretsAccountPool{
		secretsClient: secretsClient,
		shootsClient:  shootsClient,
	}
}

type secretsAccountPool struct {
	secretsClient corev1.SecretInterface
	shootsClient  gardener_apis.ShootInterface
	mux           sync.Mutex
}

func (p *secretsAccountPool) IsSubscriptionAlreadyReleased(hyperscalerType Type, tenantName string) (bool, error) {

	labelSelector := fmt.Sprintf("shared!=true, released=true, tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)

	secret, err := getK8SSecret(p.secretsClient, labelSelector)

	if err != nil {
		return false, errors.Wrapf(err, "Could not find secret used by the tenant %s and hyperscaler %s", tenantName)
	}

	if secret != nil {
		return true, nil
	}

	return false, nil
}

// just label secret as "released". This can be already marked as relesed if this step is beeing repeated
func (p *secretsAccountPool) ReleaseSubscription(hyperscalerType Type, tenantName string) error {

	released, err := p.IsSubscriptionAlreadyReleased(hyperscalerType, tenantName)

	if err != nil {
		return errors.Wrapf(err, "Could not determine if subscription for tenant %s is already releaseds: ", tenantName)
	}

	if released == true {
		return nil
	}

	p.mux.Lock()
	defer p.mux.Unlock()

	labelSelector := fmt.Sprintf("shared!=true, tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)

	secret, err := getK8SSecret(p.secretsClient, labelSelector)

	if err != nil {
		return errors.Wrapf(err, "accountPool failed to find secret used by the tenant %s and hyperscaler %s to release subscription", tenantName, hyperscalerType)
	}

	secret.Labels["released"] = "true"

	_, err = p.secretsClient.Update(secret)
	if err != nil {
		return errors.Wrapf(err, "accountPool failed to update secret with released label for tenant and hyperscaler: %s", tenantName, hyperscalerType)
	}

	return nil
}

func (p *secretsAccountPool) CountSubscriptionUsages(hyperscalerType Type, tenantName string) (int, error) {

	labelSelector := fmt.Sprintf("tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)

	secret, err := getK8SSecret(p.secretsClient, labelSelector)

	if err != nil {
		return 0, errors.Wrapf(err, "Could not find secret used by the tenant %s to count subscription usage", tenantName)
	}

	// now let's check how many shoots are using this secret
	fselector := fields.SelectorFromSet(fields.Set{fieldSecretBindingName: secret.Name}).String()

	shootlist, err := p.shootsClient.List(metav1.ListOptions{FieldSelector: fselector})

	if err != nil {
		return 0, errors.Wrapf(err, "Error while finding Gardener shoots using secret: %s", secret.Name)
	}

	if shootlist == nil || len(shootlist.Items) == 0 {
		return 0, nil
	}

	subscriptions := 0
	// count only clusters that are in good shape
	for _, s := range shootlist.Items {

		if s.Status.LastOperation != nil {
			if (s.Status.LastOperation.Type == v1beta1.LastOperationTypeCreate || s.Status.LastOperation.Type == v1beta1.LastOperationTypeReconcile || s.Status.LastOperation.Type == v1beta1.LastOperationTypeMigrate) &&
				(s.Status.LastOperation.State == v1beta1.LastOperationStateProcessing || s.Status.LastOperation.State == v1beta1.LastOperationStatePending || s.Status.LastOperation.State == v1beta1.LastOperationStateSucceeded) {
				subscriptions++
			}
		}
	}

	return subscriptions, nil
}

func (p *secretsAccountPool) Credentials(hyperscalerType Type, tenantName string) (Credentials, error) {

	labelSelector := fmt.Sprintf("tenantName=%s,hyperscalerType=%s", tenantName, hyperscalerType)
	secret, err := getK8SSecret(p.secretsClient, labelSelector)

	if err != nil {
		return Credentials{}, err
	}
	if secret != nil {
		return credentialsFromSecret(secret, hyperscalerType), nil
	}

	labelSelector = fmt.Sprintf("shared!=true, !tenantName, !released, hyperscalerType=%s", hyperscalerType)
	// lock so that only one thread can fetch an unassigned secret and assign it (update secret with tenantName)
	p.mux.Lock()
	defer p.mux.Unlock()
	secret, err = getK8SSecret(p.secretsClient, labelSelector)

	if err != nil {
		return Credentials{}, err
	}

	if secret == nil {
		return Credentials{}, errors.Errorf("accountPool failed to find unassigned secret for hyperscalerType: %s", hyperscalerType)
	}

	secret.Labels["tenantName"] = tenantName
	updatedSecret, err := p.secretsClient.Update(secret)
	if err != nil {
		return Credentials{}, errors.Wrapf(err, "accountPool error while updating secret with tenantName: %s", tenantName)
	}

	return credentialsFromSecret(updatedSecret, hyperscalerType), nil
}

func getK8SSecret(secretsClient corev1.SecretInterface, labelSelector string) (*apiv1.Secret, error) {
	secrets, err := secretsClient.List(metav1.ListOptions{
		LabelSelector: labelSelector,
	})

	if err != nil {
		return nil,
			errors.Wrapf(err, "accountPool error during secret list for LabelSelector: %s", labelSelector)
	}

	if secrets != nil && len(secrets.Items) > 0 {
		return &secrets.Items[0], nil
	}

	return nil, nil
}

func credentialsFromSecret(secret *apiv1.Secret, hyperscalerType Type) Credentials {
	return Credentials{
		Name:            secret.Name,
		HyperscalerType: hyperscalerType,
		CredentialData:  secret.Data,
	}
}
