package upgrade_kyma

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

type GetKubeconfigStep struct {
	provisionerClient   provisioner.Client
	operationManager    *process.UpgradeKymaOperationManager
	provisioningTimeout time.Duration
}

func NewGetKubeconfigStep(os storage.Operations,
	provisionerClient provisioner.Client) *GetKubeconfigStep {
	return &GetKubeconfigStep{
		provisionerClient: provisionerClient,
		operationManager:  process.NewUpgradeKymaOperationManager(os),
	}
}

var _ Step = (*GetKubeconfigStep)(nil)

func (s *GetKubeconfigStep) Name() string {
	return "Get_Kubeconfig"
}

func (s *GetKubeconfigStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if operation.Kubeconfig != "" {
		operation.InputCreator.SetKubeconfig(operation.Kubeconfig)
		return operation, 0, nil
	}
	if operation.Runtime.RuntimeID == "" {
		log.Errorf("Runtime ID is empty")
		return s.operationManager.OperationFailed(operation, "Runtime ID is empty", nil, log)
	}

	status, err := s.provisionerClient.RuntimeStatus(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.Runtime.RuntimeID)
	if err != nil {
		log.Errorf("call to provisioner RuntimeStatus failed: %s", err.Error())
		return operation, 1 * time.Minute, nil
	}

	if status.RuntimeConfiguration.Kubeconfig == nil || *status.RuntimeConfiguration.Kubeconfig == "" {
		log.Errorf("kubeconfig is not provided")
		return operation, 1 * time.Minute, nil
	}
	operation.Kubeconfig = *status.RuntimeConfiguration.Kubeconfig

	newOperation, retry, _ := s.operationManager.UpdateOperation(operation, func(operation *internal.UpgradeKymaOperation) {
		operation.Kubeconfig = *status.RuntimeConfiguration.Kubeconfig
	}, log)
	if retry > 0 {
		log.Errorf("unable to update operation")
		return operation, time.Second, nil
	}

	newOperation.InputCreator.SetKubeconfig(newOperation.Kubeconfig)

	return newOperation, 0, nil
}
