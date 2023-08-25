package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

type GetKubeconfigStep struct {
	provisionerClient   provisioner.Client
	operationManager    *process.OperationManager
	provisioningTimeout time.Duration
	k8sClientProvider   func(kubeconfig string) (client.Client, error)
}

func NewGetKubeconfigStep(os storage.Operations,
	provisionerClient provisioner.Client,
	k8sClientProvider func(kubeconfig string) (client.Client, error)) *GetKubeconfigStep {
	return &GetKubeconfigStep{
		provisionerClient: provisionerClient,
		operationManager:  process.NewOperationManager(os),
		k8sClientProvider: k8sClientProvider,
	}
}

var _ process.Step = (*GetKubeconfigStep)(nil)

func (s *GetKubeconfigStep) Name() string {
	return "Get_Kubeconfig"
}

func (s *GetKubeconfigStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {

	if operation.Kubeconfig == "" {
		if broker.IsOwnClusterPlan(operation.ProvisioningParameters.PlanID) {
			operation.Kubeconfig = operation.ProvisioningParameters.Parameters.Kubeconfig
		} else {
			if operation.RuntimeID == "" {
				log.Errorf("Runtime ID is empty")
				return s.operationManager.OperationFailed(operation, "Runtime ID is empty", nil, log)
			}
			kubeconfigFromRuntimeStatus, backoff, err := s.getKubeconfigFromRuntimeStatus(operation, log)
			if backoff > 0 {
				return operation, backoff, err
			}
			operation.Kubeconfig = kubeconfigFromRuntimeStatus
		}
	}

	return s.setK8sClientInOperation(operation, log)
}

func (s *GetKubeconfigStep) getKubeconfigFromRuntimeStatus(operation internal.Operation, log logrus.FieldLogger) (string, time.Duration, error) {

	status, err := s.provisionerClient.RuntimeStatus(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.RuntimeID)
	if err != nil {
		log.Errorf("call to provisioner RuntimeStatus failed: %s", err.Error())
		return "", 1 * time.Minute, nil
	}

	if status.RuntimeConfiguration.Kubeconfig == nil {
		log.Errorf("kubeconfig is not provided")
		return "", 1 * time.Minute, nil
	}

	kubeconfig := *status.RuntimeConfiguration.Kubeconfig

	log.Infof("kubeconfig details length: %v", len(kubeconfig))
	if len(kubeconfig) < 10 {
		log.Errorf("kubeconfig suspiciously small, requeueing after 30s")
		return "", 30 * time.Second, nil
	}

	return kubeconfig, 0, nil
}

func (s *GetKubeconfigStep) setK8sClientInOperation(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	k8sClient, err := s.k8sClientProvider(operation.Kubeconfig)
	if err != nil {
		log.Errorf("unable to create k8s client from the kubeconfig")
		return s.operationManager.RetryOperation(operation, "unable to create k8s client from the kubeconfig", err, 5*time.Second, 1*time.Minute, log)
	}
	operation.K8sClient = k8sClient
	return operation, 0, nil
}
