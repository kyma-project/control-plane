package provisioning

import (
	"context"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	apicorev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	updateSecretBackoff = 10 * time.Second
	secretName          = "sap-btp-manager"
	secretNamespace     = "kyma-system"
)

var labels = map[string]string{"app.kubernetes.io/managed-by": "kcp-kyma-environment-broker"}
var annotations = map[string]string{"Warning": "This secret is generated. Do not edit!"}

type InjectBTPOperatorCredentialsStep struct {
	operationManager  *process.OperationManager
	k8sClientProvider func(kubeconfig string) (client.Client, error)
}

func NewInjectBTPOperatorCredentialsStep(os storage.Operations, k8sClientProvider func(kcfg string) (client.Client, error)) *InjectBTPOperatorCredentialsStep {
	return &InjectBTPOperatorCredentialsStep{
		operationManager:  process.NewOperationManager(os),
		k8sClientProvider: k8sClientProvider,
	}
}

func (s *InjectBTPOperatorCredentialsStep) Name() string {
	return "Inject_BTP_Operator_Credentials"
}

func (s *InjectBTPOperatorCredentialsStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {

	if operation.RuntimeID == "" {
		log.Error("Runtime ID is empty")
		return s.operationManager.OperationFailed(operation, "Runtime ID is empty", nil, log)
	}

	if operation.K8sClient == nil {
		log.Error("kubernetes client not set")
		return s.operationManager.OperationFailed(operation, "kubernetes client not set", nil, log)
	}

	clusterID := operation.InstanceDetails.ServiceManagerClusterID
	if clusterID == "" {
		clusterID = uuid.NewString()
		updatedOperation, backoff, err := s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
			op.InstanceDetails.ServiceManagerClusterID = clusterID
		}, log)
		if err != nil {
			log.Errorf("failed to update operation: %w", err)
		}
		if backoff != 0 {
			log.Error("cannot save cluster ID")
			return updatedOperation, backoff, nil
		}
	}

	secret := s.prepareSecret(operation.ProvisioningParameters.ErsContext.SMOperatorCredentials, clusterID)

	if err := s.createOrUpdateSecret(operation.K8sClient, secret, log); err != nil {
		err = kebError.AsTemporaryError(err, "failed create/update of the secret")
		return operation, updateSecretBackoff, nil
	}
	return operation, 0, nil
}

func (s *InjectBTPOperatorCredentialsStep) prepareSecret(credentials *internal.ServiceManagerOperatorCredentials, clusterID string) *apicorev1.Secret {
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

func (s *InjectBTPOperatorCredentialsStep) createOrUpdateSecret(k8sClient client.Client, parametersBasedSecret *apicorev1.Secret, log logrus.FieldLogger) error {

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
		log.Error("failed to update the secret for BTP Manager")
		return err
	}
	log.Info("the secret for BTP Manager updated")
	return nil
}

func isNotGeneratedByKEB(secret apicorev1.Secret) bool {
	return !reflect.DeepEqual(secret.Labels, labels)
}

func updateSecretData(secret *apicorev1.Secret, secretFromParameters *apicorev1.Secret) {
	secret.StringData = secretFromParameters.StringData
	secret.ObjectMeta.Labels = labels
	secret.ObjectMeta.Annotations = annotations
}

func (s *InjectBTPOperatorCredentialsStep) createOrRetry(k8sClient client.Client, newSecret *apicorev1.Secret, err error, log logrus.FieldLogger) error {
	if apierrors.IsNotFound(err) {
		err = k8sClient.Create(context.Background(), newSecret)
		if err == nil {
			log.Info("the secret for BTP Manager created")
			return nil
		}
		log.Error("failed to create the secret for BTP Manager")
	} else {
		log.Error("failed to get the secret for BTP Manager")
	}
	return err
}
