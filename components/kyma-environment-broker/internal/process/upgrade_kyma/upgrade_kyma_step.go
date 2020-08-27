package upgrade_kyma

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
)

const UpgradeKymaTimeout = 1 * time.Hour

type UpgradeKymaStep struct {
	operationManager  *process.UpgradeKymaOperationManager
	provisionerClient provisioner.Client
}

func NewUpgradeKymaStep(os storage.Operations, cli provisioner.Client) *UpgradeKymaStep {
	return &UpgradeKymaStep{
		operationManager:  process.NewUpgradeKymaOperationManager(os),
		provisionerClient: nil,
	}
}

func (s *UpgradeKymaStep) Name() string {
	return "Upgrade_Kyma"
}

func (s *UpgradeKymaStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > UpgradeKymaTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", UpgradeKymaTimeout))
	}

	input, err := s.createUpgradeKymaInput(operation, "some-params")
	if err != nil {
		return s.operationManager.OperationFailed(operation, "invalid operation data - cannot create upgrade_kyma input")
	}

	var provisionerResponse gqlschema.OperationStatus
	if operation.ProvisionerOperationID == "" {
		provisionerResponse, err := s.provisionerClient.UpgradeRuntime("GLOBAL_ACCOUNT_ID", "SUBACCOUNT_ID", input)
		if err != nil {
			log.Errorf("call to provisioner failed: %s", err)
			return operation, 5 * time.Second, nil
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
		provisionerResponse, err = s.provisionerClient.RuntimeOperationStatus("GLOBAL_ACCOUNT_ID", operation.ProvisionerOperationID)
		if err != nil {
			log.Errorf("call to provisioner about operation status failed: %s", err)
			return operation, 1 * time.Minute, nil
		}
	}
	if provisionerResponse.RuntimeID == nil {
		return operation, 1 * time.Minute, nil
	}
	log = log.WithField("runtimeID", *provisionerResponse.RuntimeID)
	log.Infof("call to provisioner succeeded", *provisionerResponse.RuntimeID)

}

func (s *UpgradeKymaStep) createUpgradeKymaInput(operation internal.UpgradeKymaOperation, parameters string) (gqlschema.UpgradeRuntimeInput, error) {
	var request gqlschema.UpgradeRuntimeInput
	// TODO: implement InputProvider or extend existing one and use it

	return request, nil
}
