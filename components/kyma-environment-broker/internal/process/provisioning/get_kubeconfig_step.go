package provisioning

import (
	"crypto/sha256"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

type GetKubeconfigStep struct {
	provisionerClient   provisioner.Client
	operationManager    *process.ProvisionOperationManager
	provisioningTimeout time.Duration
}

func NewGetKubeconfigStep(os storage.Operations,
	provisionerClient provisioner.Client) *GetKubeconfigStep {
	return &GetKubeconfigStep{
		provisionerClient: provisionerClient,
		operationManager:  process.NewProvisionOperationManager(os),
	}
}

var _ Step = (*GetKubeconfigStep)(nil)

func (s *GetKubeconfigStep) Name() string {
	return "Get_Kubeconfig"
}

func (s *GetKubeconfigStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	// TODO: check for KUBECONFIG from input parameters
	if operation.Kubeconfig != "" {
		return operation, 0, nil
	}

	if operation.ProvisioningParameters.Parameters.Kubeconfig != "" {
		operation.Kubeconfig = operation.ProvisioningParameters.Parameters.Kubeconfig

		newOperation, retry, _ := s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
			operation.Kubeconfig = operation.ProvisioningParameters.Parameters.Kubeconfig
		}, log)

		if retry > 0 {
			log.Errorf("unable to update operation")
			return operation, time.Second, nil
		}

		return newOperation, 0, nil
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
	k := *status.RuntimeConfiguration.Kubeconfig
	hash := sha256.Sum256([]byte(k))
	log.Infof("kubeconfig details length: %v, sha256: %v", len(k), string(hash[:]))
	if len(k) < 10 {
		log.Errorf("kubeconfig suspiciously small, requeueing after 30s")
		return operation, 30 * time.Second, nil
	}
	operation.Kubeconfig = *status.RuntimeConfiguration.Kubeconfig

	newOperation, retry, _ := s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
		operation.Kubeconfig = *status.RuntimeConfiguration.Kubeconfig
	}, log)
	if retry > 0 {
		log.Errorf("unable to update operation")
		return operation, time.Second, nil
	}

	return newOperation, 0, nil
}
