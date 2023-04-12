package btpoperatorcredentials

import (
	"context"
	"fmt"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/sirupsen/logrus"
	apicorev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	Keb                       = "kcp-kyma-environment-broker"
	BtpManagerSecretName      = "sap-btp-manager"
	BtpManagerSecretNamespace = "kyma-system"
)

var (
	BtpManagerLabels      = map[string]string{"app.kubernetes.io/managed-by": Keb, "app.kubernetes.io/watched-by": Keb}
	BtpManagerAnnotations = map[string]string{"Warning": "This secret is generated. Do not edit!"}
)

type BTPOperatorHandler struct{}

func (s *BTPOperatorHandler) CreateOrUpdateSecret(k8sClient client.Client, parametersBasedSecret *apicorev1.Secret, log logrus.FieldLogger) error {
	clusterSecret := apicorev1.Secret{}
	err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: BtpManagerSecretNamespace, Name: BtpManagerSecretName}, &clusterSecret)
	if err != nil {
		return s.createOrRetry(k8sClient, parametersBasedSecret, err, log)
	}
	if isNotGeneratedByKEB(clusterSecret) {
		log.Warnf("the secret %s was not created by KEB and its data will be overwritten", BtpManagerSecretName)
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
			Name:        BtpManagerSecretName,
			Namespace:   BtpManagerSecretNamespace,
			Labels:      BtpManagerLabels,
			Annotations: BtpManagerAnnotations,
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
		namespace := &apicorev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: BtpManagerSecretNamespace}}
		err = k8sClient.Create(context.Background(), namespace)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			log.Warnf("could not create %s namespace: %s", BtpManagerSecretNamespace, err)
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

func CompareSecretsData(current, expected *apicorev1.Secret) (bool, error) {
	var errors []string

	tryGet := func(sd map[string]string, key string, res *[]string) {
		obj, ok := sd[key]
		if !ok {
			errors = append(errors, fmt.Sprintf(""))
		}
		*res = append(*res, obj)
	}

	getData := func(sd map[string]string) []string {
		var res []string
		tryGet(sd, "clientid", &res)
		tryGet(sd, "clientsecret", &res)
		tryGet(sd, "sm_url", &res)
		tryGet(sd, "tokenurl", &res)
		tryGet(sd, "cluster_id", &res)
		return res
	}

	currentData := getData(current.StringData)
	expectedData := getData(expected.StringData)
	return reflect.DeepEqual(currentData, expectedData), fmt.Errorf("%s", strings.Join(errors, ","))
}

func isNotGeneratedByKEB(secret apicorev1.Secret) bool {
	return !reflect.DeepEqual(secret.Labels, BtpManagerLabels)
}

func updateSecretData(secret *apicorev1.Secret, secretFromParameters *apicorev1.Secret) {
	secret.StringData = secretFromParameters.StringData
	secret.ObjectMeta.Labels = BtpManagerLabels
	secret.ObjectMeta.Annotations = BtpManagerAnnotations
}
