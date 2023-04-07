package btpoperatorcredentials

import (
	"context"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/sirupsen/logrus"
	apicorev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	keb             = "kcp-kyma-environment-broker"
	secretName      = "sap-btp-manager"
	secretNamespace = "kyma-system"
)

var labels = map[string]string{"app.kubernetes.io/managed-by": keb, "app.kubernetes.io/watched-by": keb}
var annotations = map[string]string{"Warning": "This secret is generated. Do not edit!"}

type BTPOperatorHandler struct{}

func (s *BTPOperatorHandler) CreateOrUpdateSecret(k8sClient client.Client, parametersBasedSecret *apicorev1.Secret, log logrus.FieldLogger) error {
	clusterSecret := apicorev1.Secret{}
	err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: secretNamespace, Name: secretName}, &clusterSecret)
	if err != nil {
		return s.createOrRetry(k8sClient, parametersBasedSecret, err, log)
	}
	if isNotGeneratedByKEB(clusterSecret) {
		log.Warnf("the secret %s was not created by KEB and its data will be overwritten", secretName)
	}
	updateSecretData(&clusterSecret, parametersBasedSecret)
	err = k8sClient.Update(context.Background(), &clusterSecret)
	if err != nil {
		log.Errorf("failed to update the secret for BTP Manager: %s", err)
		return err
	}
	log.Info("the secret for BTP Manager updated")
	return nil
}

func (s *BTPOperatorHandler) PrepareSecret(credentials *internal.ServiceManagerOperatorCredentials, clusterID string) *apicorev1.Secret {
	return &apicorev1.Secret{
		TypeMeta: v1.TypeMeta{Kind: "Secret"},
		ObjectMeta: v1.ObjectMeta{
			Name:        secretName,
			Namespace:   secretNamespace,
			Labels:      labels,
			Annotations: annotations,
		},
		StringData: map[string]string{
			"clientid":     credentials.ClientID,
			"clientsecret": credentials.ClientSecret,
			"sm_url":       credentials.ServiceManagerURL,
			"tokenurl":     credentials.URL,
			"cluster_id":   clusterID},
		Type: apicorev1.SecretTypeOpaque,
	}
}

func (s *BTPOperatorHandler) createOrRetry(k8sClient client.Client, newSecret *apicorev1.Secret, err error, log logrus.FieldLogger) error {
	if apierrors.IsNotFound(err) {
		namespace := &apicorev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: secretNamespace}}
		err = k8sClient.Create(context.Background(), namespace)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			log.Warnf("could not create %s namespace: %s", secretNamespace, err)
			return err
		}

		err = k8sClient.Create(context.Background(), newSecret)
		if err == nil {
			log.Info("the secret for BTP Manager created")
			return nil
		}
		log.Errorf("failed to create the secret for BTP Manager: %s", err)
	} else {
		log.Errorf("failed to get the secret for BTP Manager: %s", err)
	}
	return err
}

func CompareContentFromSkr(secret *apicorev1.Secret, obj client.Object) bool {
	return true
}

func isNotGeneratedByKEB(secret apicorev1.Secret) bool {
	return !reflect.DeepEqual(secret.Labels, labels)
}

func updateSecretData(secret *apicorev1.Secret, secretFromParameters *apicorev1.Secret) {
	secret.StringData = secretFromParameters.StringData
	secret.ObjectMeta.Labels = labels
	secret.ObjectMeta.Annotations = annotations
}
