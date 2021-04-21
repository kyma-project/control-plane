package provisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"

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
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", CreateRuntimeTimeout), log)
	}
	requestInput, err := s.createProvisionInput(operation)
	if err != nil {
		log.Errorf("Unable to create provisioning input: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "invalid operation data - cannot create provisioning input", log)
	}

	var provisionerResponse gqlschema.OperationStatus
	if operation.ProvisionerOperationID == "" {
		log.Infof("call ProvisionRuntime: kymaVersion=%s, kubernetesVersion=%s, region=%s, kymaProfile=%s, provider=%s, name=%s",
			requestInput.KymaConfig.Version,
			requestInput.ClusterConfig.GardenerConfig.KubernetesVersion,
			requestInput.ClusterConfig.GardenerConfig.Region,
			requestInput.KymaConfig.Profile,
			requestInput.ClusterConfig.GardenerConfig.Provider,
			requestInput.ClusterConfig.GardenerConfig.Name)

		provisionerResponse, err := s.provisionerClient.ProvisionRuntime(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.ProvisioningParameters.ErsContext.SubAccountID, requestInput)
		switch {
		case kebError.IsTemporaryError(err):
			log.Errorf("call to provisioner failed (temporary error): %s", err)
			return operation, 5 * time.Second, nil
		case err != nil:
			log.Errorf("call to Provisioner failed: %s", err)
			return s.operationManager.OperationFailed(operation, "call to the provisioner service failed", log)
		}

		repeat := time.Duration(0)
		operation, repeat = s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
			operation.ProvisionerOperationID = *provisionerResponse.ID
			if provisionerResponse.RuntimeID != nil {
				operation.RuntimeID = *provisionerResponse.RuntimeID
			}
		}, log)
		if repeat != 0 {
			log.Errorf("cannot save operation ID from provisioner")
			return operation, 5 * time.Second, nil
		}
	}

	if provisionerResponse.RuntimeID == nil {
		provisionerResponse, err = s.provisionerClient.RuntimeOperationStatus(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.ProvisionerOperationID)
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
		log.Errorf("cannot insert runtimeState: %s", err)
		return operation, 10 * time.Second, nil
	}

	err = s.updateInstance(operation.InstanceID, *provisionerResponse.RuntimeID, requestInput.ClusterConfig.GardenerConfig.Region)
	switch {
	case err == nil:
	case dberr.IsConflict(err):
		err := s.updateInstance(operation.InstanceID, *provisionerResponse.RuntimeID, requestInput.ClusterConfig.GardenerConfig.Region)
		if err != nil {
			log.Errorf("cannot update instance: %s", err)
			return operation, 1 * time.Minute, nil
		}
	}

	log.Info("runtime creation process initiated successfully")
	// return repeat mode (1 sec) to start the initialization step which will now check the runtime status
	return operation, 1 * time.Second, nil
}

func (s *CreateRuntimeStep) updateInstance(id, runtimeID, region string) error {
	instance, err := s.instanceStorage.GetByID(id)
	if err != nil {
		return errors.Wrap(err, "while getting instance")
	}
	instance.RuntimeID = runtimeID
	instance.ProviderRegion = region
	_, err = s.instanceStorage.Update(*instance)
	if err != nil {
		return errors.Wrap(err, "while updating instance")
	}

	return nil
}

func (s *CreateRuntimeStep) createProvisionInput(operation internal.ProvisioningOperation) (gqlschema.ProvisionRuntimeInput, error) {
	var request gqlschema.ProvisionRuntimeInput

	operation.InputCreator.SetProvisioningParameters(operation.ProvisioningParameters)
	operation.InputCreator.SetShootName(operation.ShootName)
	operation.InputCreator.SetLabel(brokerKeyPrefix+"instance_id", operation.InstanceID)
	operation.InputCreator.SetLabel(globalKeyPrefix+"subaccount_id", operation.ProvisioningParameters.ErsContext.SubAccountID)
	request, err := operation.InputCreator.CreateProvisionRuntimeInput()
	if err != nil {
		return request, errors.Wrap(err, "while building input for provisioner")
	}

	return request, nil
}
