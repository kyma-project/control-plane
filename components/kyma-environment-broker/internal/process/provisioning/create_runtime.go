package provisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// the time after which the operation is marked as expired
	CreateRuntimeTimeout = 1 * time.Hour

	brokerKeyPrefix = "broker_"
	globalKeyPrefix = "global_"
)

type CreateRuntimeStep struct {
	operationManager    *process.ProvisionOperationManager
	instanceStorage     storage.Instances
	runtimeStateStorage storage.RuntimeStates
	provisionerClient   provisioner.Client
}

func NewCreateRuntimeStep(os storage.Operations, runtimeStorage storage.RuntimeStates, is storage.Instances, cli provisioner.Client) *CreateRuntimeStep {
	return &CreateRuntimeStep{
		operationManager:    process.NewProvisionOperationManager(os),
		instanceStorage:     is,
		provisionerClient:   cli,
		runtimeStateStorage: runtimeStorage,
	}
}

func (s *CreateRuntimeStep) Name() string {
	return "Create_Runtime"
}

func (s *CreateRuntimeStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > CreateRuntimeTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", CreateRuntimeTimeout))
	}

	pp, err := operation.GetProvisioningParameters()
	if err != nil {
		log.Errorf("Unable to get provisioning parameters: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "invalid operation provisioning parameters")
	}

	requestInput, err := s.createProvisionInput(operation, pp)
	if err != nil {
		log.Errorf("Unable to create provisioning input: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "invalid operation data - cannot create provisioning input")
	}

	var provisionerResponse gqlschema.OperationStatus
	if operation.ProvisionerOperationID == "" {
		log.Infof("call ProvisionRuntime: kymaVersion=%s, kubernetesVersion=%s", requestInput.KymaConfig.Version, requestInput.ClusterConfig.GardenerConfig.KubernetesVersion)
		provisionerResponse, err := s.provisionerClient.ProvisionRuntime(pp.ErsContext.GlobalAccountID, pp.ErsContext.SubAccountID, requestInput)
		switch {
		case kebError.IsTemporaryError(err):
			log.Errorf("call to provisioner failed (temporary error): %s", err)
			return operation, 5 * time.Second, nil
		case err != nil:
			log.Errorf("call to Provisioner failed: %s", err)
			return s.operationManager.OperationFailed(operation, "invalid operation data - cannot create provisioning input")
		}

		operation.ProvisionerOperationID = *provisionerResponse.ID
		if provisionerResponse.RuntimeID != nil {
			operation.RuntimeID = *provisionerResponse.RuntimeID
		}
		operation, repeat := s.operationManager.UpdateOperation(operation)
		if repeat != 0 {
			log.Errorf("cannot save operation ID from provisioner")
			return operation, 5 * time.Second, nil
		}
	}

	if provisionerResponse.RuntimeID == nil {
		provisionerResponse, err = s.provisionerClient.RuntimeOperationStatus(pp.ErsContext.GlobalAccountID, operation.ProvisionerOperationID)
		if err != nil {
			log.Errorf("call to provisioner about operation status failed: %s", err)
			return operation, 1 * time.Minute, nil
		}
	}
	if provisionerResponse.RuntimeID == nil {
		return operation, 1 * time.Minute, nil
	}
	log = log.WithField("runtimeID", *provisionerResponse.RuntimeID)
	log.Infof("call to provisioner succeeded, got operation ID %q", *provisionerResponse.ID)

	err = s.runtimeStateStorage.Insert(
		internal.NewRuntimeState(*provisionerResponse.RuntimeID, operation.ID, requestInput.KymaConfig, requestInput.ClusterConfig.GardenerConfig),
	)
	if err != nil {
		return operation, 10 * time.Second, nil
	}

	instance, err := s.instanceStorage.GetByID(operation.InstanceID)
	if err != nil {
		log.Errorf("cannot get instance: %s", err)
		return operation, 1 * time.Minute, nil
	}
	instance.RuntimeID = *provisionerResponse.RuntimeID
	// Save provider region in instance ProvivisiongParameters so that all instances store it regardless of ServicePlan, and whether region parameter was input or not
	pp.Parameters.Region = &requestInput.ClusterConfig.GardenerConfig.Region
	err = instance.SetProvisioningParameters(pp)
	if err != nil {
		return s.operationManager.OperationFailed(operation, "invalid provisioning parameters to store in instance")
	}

	err = s.instanceStorage.Update(*instance)
	if err != nil {
		log.Errorf("cannot update instance in storage: %s", err)
		return operation, 10 * time.Second, nil
	}

	log.Info("runtime creation process initiated successfully")
	// return repeat mode (1 sec) to start the initialization step which will now check the runtime status
	return operation, 1 * time.Second, nil
}

func (s *CreateRuntimeStep) createProvisionInput(operation internal.ProvisioningOperation, parameters internal.ProvisioningParameters) (gqlschema.ProvisionRuntimeInput, error) {
	var request gqlschema.ProvisionRuntimeInput

	operation.InputCreator.SetProvisioningParameters(parameters)
	operation.InputCreator.SetLabel(brokerKeyPrefix+"instance_id", operation.InstanceID)
	operation.InputCreator.SetLabel(globalKeyPrefix+"subaccount_id", parameters.ErsContext.SubAccountID)
	request, err := operation.InputCreator.CreateProvisionRuntimeInput()
	if err != nil {
		return request, errors.Wrap(err, "while building input for provisioner")
	}

	return request, nil
}
