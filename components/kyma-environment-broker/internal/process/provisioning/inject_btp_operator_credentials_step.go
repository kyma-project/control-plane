package provisioning

import (
	"time"

	btpmanagercredentials "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/btpmanager/credentials"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	updateSecretBackoff = 10 * time.Second
)

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
			log.Errorf("failed to update operation: %s", err)
		}
		if backoff != 0 {
			log.Error("cannot save cluster ID")
			return updatedOperation, backoff, nil
		}
	}

	secret, err := btpmanagercredentials.PrepareSecret(operation.ProvisioningParameters.ErsContext.SMOperatorCredentials, clusterID)
	if err != nil {
		return s.operationManager.OperationFailed(operation, "secret preparation failed", err, log)
	}

	if err := btpmanagercredentials.CreateOrUpdateSecret(operation.K8sClient, secret, log); err != nil {
		err = kebError.AsTemporaryError(err, "failed create/update of the secret")
		return operation, updateSecretBackoff, nil
	}
	return operation, 0, nil
}
