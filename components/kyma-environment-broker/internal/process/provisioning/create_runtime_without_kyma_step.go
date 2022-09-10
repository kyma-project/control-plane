package provisioning

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
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

type CreateRuntimeWithoutKymaStep struct {
	operationManager    *process.ProvisionOperationManager
	instanceStorage     storage.Instances
	runtimeStateStorage storage.RuntimeStates
	provisionerClient   provisioner.Client
}

func NewCreateRuntimeWithoutKymaStep(os storage.Operations, runtimeStorage storage.RuntimeStates, is storage.Instances, cli provisioner.Client) *CreateRuntimeWithoutKymaStep {
	return &CreateRuntimeWithoutKymaStep{
		operationManager:    process.NewProvisionOperationManager(os),
		instanceStorage:     is,
		provisionerClient:   cli,
		runtimeStateStorage: runtimeStorage,
	}
}

func (s *CreateRuntimeWithoutKymaStep) Name() string {
	return "Create_Runtime_Without_Kyma"
}

func (s *CreateRuntimeWithoutKymaStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if operation.RuntimeID != "" {
		log.Infof("RuntimeID already set %s, skipping", operation.RuntimeID)
		return operation, 0, nil
	}
	if time.Since(operation.UpdatedAt) > CreateRuntimeTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", CreateRuntimeTimeout), nil, log)
	}

	requestInput, err := s.createProvisionInput(operation)
	if err != nil {
		log.Errorf("Unable to create provisioning input: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "invalid operation data - cannot create provisioning input", err, log)
	}

	if requestInput.ClusterConfig.GardenerConfig != nil {
		log.Infof("call ProvisionRuntime: kubernetesVersion=%s, region=%s, provider=%s, name=%s",
			requestInput.ClusterConfig.GardenerConfig.KubernetesVersion,
			requestInput.ClusterConfig.GardenerConfig.Region,
			requestInput.ClusterConfig.GardenerConfig.Provider,
			requestInput.ClusterConfig.GardenerConfig.Name)
	}

	var runtimeId string
	var provisionerOperationId string
	if operation.ProvisioningParameters.PlanID == broker.OwnClusterPlanID {

		runtimeId = uuid.New().String()
		operation.RuntimeID = runtimeId
		operation.ProvisionerOperationID = ""

	} else {

		provisionerResponse, err := s.provisionerClient.ProvisionRuntime(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.ProvisioningParameters.ErsContext.SubAccountID, requestInput)
		switch {
		case kebError.IsTemporaryError(err):
			log.Errorf("call to provisioner failed (temporary error): %s", err)
			return operation, 5 * time.Second, nil
		case err != nil:
			log.Errorf("call to Provisioner failed: %s", err)
			return s.operationManager.OperationFailed(operation, "call to the provisioner service failed", err, log)
		}
		log.Infof("Provisioning runtime in the Provisioner started, RuntimeID=%s", *provisionerResponse.RuntimeID)
		runtimeId = *provisionerResponse.RuntimeID
		provisionerOperationId = *provisionerResponse.ID
	}

	repeat := time.Duration(0)
	operation, repeat, _ = s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
		operation.ProvisionerOperationID = provisionerOperationId
		if runtimeId != "" {
			operation.RuntimeID = runtimeId
		}
	}, log)
	if repeat != 0 {
		log.Errorf("cannot save operation ID from provisioner")
		return operation, 5 * time.Second, nil
	}

	// todo: dop we still need this
	err = s.runtimeStateStorage.Insert(
		internal.NewRuntimeState(runtimeId, operation.ID, requestInput.KymaConfig, requestInput.ClusterConfig.GardenerConfig),
	)
	if err != nil {
		log.Errorf("cannot insert runtimeState: %s", err)
		return operation, 10 * time.Second, nil
	}

	region := ""
	if requestInput.ClusterConfig.GardenerConfig != nil {
		region = requestInput.ClusterConfig.GardenerConfig.Region
	}

	err = s.updateInstance(operation.InstanceID,
		runtimeId,
		region)

	switch {
	case err == nil:
	case dberr.IsConflict(err):
		err := s.updateInstance(operation.InstanceID, runtimeId, region)
		if err != nil {
			log.Errorf("cannot update instance: %s", err)
			return operation, 1 * time.Minute, nil
		}
	}

	log.Info("runtime creation process initiated successfully")
	return operation, 0, nil
}

func (s *CreateRuntimeWithoutKymaStep) updateInstance(id, runtimeID, region string) error {
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

func (s *CreateRuntimeWithoutKymaStep) createProvisionInput(operation internal.ProvisioningOperation) (gqlschema.ProvisionRuntimeInput, error) {
	operation.InputCreator.SetProvisioningParameters(operation.ProvisioningParameters)
	operation.InputCreator.SetShootName(operation.ShootName)
	operation.InputCreator.SetShootDomain(operation.ShootDomain)
	operation.InputCreator.SetShootDNSProviders(operation.ShootDNSProviders)
	operation.InputCreator.SetLabel(brokerKeyPrefix+"instance_id", operation.InstanceID)
	operation.InputCreator.SetLabel(globalKeyPrefix+"subaccount_id", operation.ProvisioningParameters.ErsContext.SubAccountID)
	operation.InputCreator.SetLabel(grafanaURLLabel, fmt.Sprintf("https://grafana.%s", operation.ShootDomain))
	request, err := operation.InputCreator.CreateProvisionClusterInput()
	if err != nil {
		return request, errors.Wrap(err, "while building input for provisioner")
	}

	if request.ClusterConfig.GardenerConfig != nil {
		request.ClusterConfig.GardenerConfig.ShootNetworkingFilterDisabled = operation.ProvisioningParameters.ErsContext.DisableEnterprisePolicyFilter()
	}

	return request, nil
}
