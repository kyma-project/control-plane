package deprovisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GetKubeconfigStep struct {
	provisionerClient   provisioner.Client
	operationManager    *process.DeprovisionOperationManager
	provisioningTimeout time.Duration
	k8sClientProvider   func(kcfg string) (client.Client, error)
}

func NewGetKubeconfigStep(os storage.Operations, provisionerClient provisioner.Client, k8sClientProvider func(kcfg string) (client.Client, error)) *GetKubeconfigStep {
	return &GetKubeconfigStep{
		provisionerClient: provisionerClient,
		operationManager:  process.NewDeprovisionOperationManager(os),
		k8sClientProvider: k8sClientProvider,
	}
}

func (s *GetKubeconfigStep) Name() string {
	return "Get_Kubeconfig"
}

func (s *GetKubeconfigStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if operation.IsServiceInstanceDeleted {
		return operation, 0, nil
	}

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
		log.Infof("RuntimeID is empty, skipping step")
		operation.IsServiceInstanceDeleted = true
		return operation, 0, nil
	}

	status, err := s.provisionerClient.RuntimeStatus(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.RuntimeID)
	if err != nil {
		return handleError(s.Name(), operation, err, log, "call to provisioner RuntimeStatus failed")
	}

	if status.RuntimeConfiguration.Kubeconfig == nil || *status.RuntimeConfiguration.Kubeconfig == "" {
		log.Infof("kubeconfig is not provided, skipping step")
		operation.IsServiceInstanceDeleted = true
		return operation, 0, nil
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
