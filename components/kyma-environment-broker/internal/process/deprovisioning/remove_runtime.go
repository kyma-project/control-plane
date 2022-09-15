package deprovisioning

import (
	"fmt"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

type RemoveRuntimeStep struct {
	operationManager   *process.OperationManager
	instanceStorage    storage.Instances
	provisionerClient  provisioner.Client
	provisionerTimeout time.Duration
}

func NewRemoveRuntimeStep(os storage.Operations, is storage.Instances, cli provisioner.Client, provisionerTimeout time.Duration) *RemoveRuntimeStep {
	return &RemoveRuntimeStep{
		operationManager:   process.NewOperationManager(os),
		instanceStorage:    is,
		provisionerClient:  cli,
		provisionerTimeout: provisionerTimeout,
	}
}

func (s *RemoveRuntimeStep) Name() string {
	return "Remove_Runtime"
}

func (s *RemoveRuntimeStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > s.provisionerTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", s.provisionerTimeout), nil, log)
	}
	if operation.RuntimeID == "" || operation.ProvisioningParameters.PlanID == broker.OwnClusterPlanID {
		// happens when provisioning process failed and Create_Runtime step was never reached
		// It can also happen when the SKR is suspended (technically deprovisioned)
		log.Infof("Runtime does not exist for instance id %q", operation.InstanceID)

		err := s.cleanUp(&operation, log)
		if err != nil {
			return operation, 1 * time.Second, nil
		}
		operation, _, _ := s.operationManager.OperationSucceeded(operation, "Runtime was never provisioned", log)
		return operation, 1 * time.Second, nil
	}

	if operation.ProvisionerOperationID == "" {
		provisionerResponse, err := s.provisionerClient.DeprovisionRuntime(operation.GlobalAccountID, operation.RuntimeID)
		if err != nil {
			log.Errorf("unable to deprovision runtime: %s", err)
			return operation, 10 * time.Second, nil
		}
		log.Infof("fetched ProvisionerOperationID=%s", provisionerResponse)
		repeat := time.Duration(0)
		operation, repeat, _ = s.operationManager.UpdateOperation(operation, func(operation *internal.Operation) {
			operation.ProvisionerOperationID = provisionerResponse
		}, log)
		if repeat != 0 {
			return operation, 5 * time.Second, nil
		}
	}

	log.Infof("runtime deletion process initiated successfully")
	// return repeat mode (1 sec) to start the initialization step which will now check the runtime status
	return operation, 1 * time.Second, nil
}

func (s *RemoveRuntimeStep) cleanUp(operation *internal.Operation, log logrus.FieldLogger) error {
	if !operation.Temporary {
		log.Info("Removing the instance")
		err := s.instanceStorage.Delete(operation.InstanceID)
		if err != nil {
			return err
		}
		log.Info("Removing the userID field from operation")
		operation.ProvisioningParameters.ErsContext.UserID = ""
	}
	return nil
}
