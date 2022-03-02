package update

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GetKubeconfigStep struct {
	provisionerClient   provisioner.Client
	operationManager    *process.UpdateOperationManager
	provisioningTimeout time.Duration
	k8sClientProvider   func(kcfg string) (client.Client, error)
}

func NewGetKubeconfigStep(os storage.Operations, provisionerClient provisioner.Client, k8sClientProvider func(kcfg string) (client.Client, error)) *GetKubeconfigStep {
	return &GetKubeconfigStep{
		provisionerClient: provisionerClient,
		operationManager:  process.NewUpdateOperationManager(os),
		k8sClientProvider: k8sClientProvider,
	}
}

var _ Step = (*GetKubeconfigStep)(nil)

func (s *GetKubeconfigStep) Name() string {
	return "Get_Kubeconfig"
}

func (s *GetKubeconfigStep) Run(operation internal.UpdatingOperation, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	if operation.Kubeconfig != "" {
		cli, err := s.k8sClientProvider(operation.Kubeconfig)
		if err != nil {
			log.Errorf("Unable to create k8s client from the kubeconfig")
			return s.operationManager.OperationFailed(operation, "could not create a k8s client", err, log)
		}
		operation.K8sClient = cli
		return operation, 0, nil
	}
	if operation.RuntimeID == "" {
		log.Errorf("Runtime ID is empty")
		return s.operationManager.OperationFailed(operation, "Runtime ID is empty", nil, log)
	}

	status, err := s.provisionerClient.RuntimeStatus(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.RuntimeID)
	if err != nil {
		log.Errorf("call to provisioner RuntimeStatus failed: %s", err.Error())
		return operation, 1 * time.Minute, nil
	}

	if status.RuntimeConfiguration.Kubeconfig == nil {
		log.Errorf("kubeconfig is not provided")
		return operation, 1 * time.Minute, nil
	}
	cli, err := s.k8sClientProvider(*status.RuntimeConfiguration.Kubeconfig)
	if err != nil {
		log.Errorf("Unable to create k8s client from the kubeconfig")
		return s.operationManager.OperationFailed(operation, "could not create a k8s client", err, log)
	}
	operation.Kubeconfig = *status.RuntimeConfiguration.Kubeconfig
	operation.K8sClient = cli

	return operation, 0, nil
}
