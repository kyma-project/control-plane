package provisioning

import (
	"crypto/sha256"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

type GetKubeconfigStep struct {
	provisionerClient   provisioner.Client
	operationManager    *process.OperationManager
	provisioningTimeout time.Duration
}

func NewGetKubeconfigStep(os storage.Operations,
	provisionerClient provisioner.Client) *GetKubeconfigStep {
	return &GetKubeconfigStep{
		provisionerClient: provisionerClient,
		operationManager:  process.NewOperationManager(os),
	}
}

var _ process.Step = (*GetKubeconfigStep)(nil)

func (s *GetKubeconfigStep) Name() string {
	return "Get_Kubeconfig"
}

func (s *GetKubeconfigStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if operation.Kubeconfig != "" {
		return operation, 0, nil
	}

	if operation.ProvisioningParameters.PlanID == broker.OwnClusterPlanID {

		newOperation, backoff, _ := s.operationManager.UpdateOperation(operation, func(operation *internal.Operation) {
			operation.Kubeconfig = operation.ProvisioningParameters.Parameters.Kubeconfig
		}, log)

		if backoff > 0 {
			log.Errorf("unable to update operation")
			return operation, backoff, nil
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

	newOperation, backoff, _ := s.operationManager.UpdateOperation(operation, func(operation *internal.Operation) {
		operation.Kubeconfig = *status.RuntimeConfiguration.Kubeconfig
	}, log)

	if backoff > 0 {
		log.Errorf("unable to update operation")
		return operation, backoff, nil
	}

	return newOperation, 0, nil
}
