package upgrade_kyma

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
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

	pp, err := operation.GetProvisioningParameters()
	if err != nil {
		return s.operationManager.OperationFailed(operation, "invalid operation provisioning parameters")
	}


	requestInput, err := s.createUpgradeKymaInput(operation, pp)
	if err != nil {
		return s.operationManager.OperationFailed(operation, "invalid operation data - cannot create upgradeKyma input")
	}

	var provisionerResponse gqlschema.OperationStatus
	if operation.ProvisionerOperationID == "" {
		provisionerResponse, err := s.provisionerClient.UpgradeRuntime(pp.ErsContext.GlobalAccountID, pp.ErsContext.SubAccountID, requestInput)
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


	log = log.WithField("runtimeID", *provisionerResponse.RuntimeID)
	log.Infof("call to provisioner succeeded", *provisionerResponse.RuntimeID)

	log.Infof("kymaUpgrade for runtime id %q process initiated successfully", *provisionerResponse.RuntimeID)
	// return repeat mode (1 sec) to start the initialization step which will now check the runtime status
	return operation, 1 * time.Second, nil
}

func (s *UpgradeKymaStep) createUpgradeKymaInput(operation internal.UpgradeKymaOperation, parameters internal.ProvisioningParameters) (gqlschema.UpgradeRuntimeInput, error) {
	var request gqlschema.UpgradeRuntimeInput


	request, err := operation.InputCreator.Create()
	if err != nil {
		return request, errors.Wrap(err, "while building upgradeRuntimeInput for provisioner")
	}

	return request, nil
}
