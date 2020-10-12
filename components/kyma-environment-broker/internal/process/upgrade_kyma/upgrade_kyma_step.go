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

type UpgradeKymaStep struct {
	operationManager    *process.UpgradeKymaOperationManager
	provisionerClient   provisioner.Client
	runtimeStateStorage storage.RuntimeStates
	timeSchedule        TimeSchedule
}

func NewUpgradeKymaStep(os storage.Operations, runtimeStorage storage.RuntimeStates, cli provisioner.Client, timeSchedule *TimeSchedule) *UpgradeKymaStep {
	ts := timeSchedule
	if ts == nil {
		ts = &TimeSchedule{
			Retry:              5 * time.Second,
			StatusCheck:        time.Minute,
			UpgradeKymaTimeout: time.Hour,
		}
	}
	return &UpgradeKymaStep{
		operationManager:    process.NewUpgradeKymaOperationManager(os),
		provisionerClient:   cli,
		runtimeStateStorage: runtimeStorage,
		timeSchedule:        *ts,
	}
}

func (s *UpgradeKymaStep) Name() string {
	return "Upgrade_Kyma"
}

func (s *UpgradeKymaStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > s.timeSchedule.UpgradeKymaTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", s.timeSchedule.UpgradeKymaTimeout))
	}

	pp, err := operation.GetProvisioningParameters()
	if err != nil {
		return s.operationManager.OperationFailed(operation, "invalid operation provisioning parameters")
	}

	requestInput, err := s.createUpgradeKymaInput(operation)
	if err != nil {
		return s.operationManager.OperationFailed(operation, "invalid operation data - cannot create upgradeKyma input")
	}

	if operation.DryRun {
		return s.operationManager.OperationSucceeded(operation, "dry run succeeded")
	}

	var provisionerResponse gqlschema.OperationStatus
	if operation.ProvisionerOperationID == "" {
		// trigger upgradeRuntime mutation
		provisionerResponse, err := s.provisionerClient.UpgradeRuntime(pp.ErsContext.GlobalAccountID, operation.RuntimeID, requestInput)
		if err != nil {
			log.Errorf("call to provisioner failed: %s", err)
			return operation, s.timeSchedule.Retry, nil
		}
		operation.ProvisionerOperationID = *provisionerResponse.ID

		operation, repeat := s.operationManager.UpdateOperation(operation)
		if repeat != 0 {
			log.Errorf("cannot save operation ID from provisioner")
			return operation, s.timeSchedule.Retry, nil
		}
	}

	if provisionerResponse.RuntimeID == nil {
		provisionerResponse, err = s.provisionerClient.RuntimeOperationStatus(pp.ErsContext.GlobalAccountID, operation.ProvisionerOperationID)
		if err != nil {
			log.Errorf("call to provisioner about operation status failed: %s", err)
			return operation, s.timeSchedule.Retry, nil
		}
	}
	if provisionerResponse.RuntimeID == nil {
		return operation, s.timeSchedule.StatusCheck, nil
	}
	log = log.WithField("runtimeID", *provisionerResponse.RuntimeID)
	log.Infof("call to provisioner succeeded, got operation ID %q", *provisionerResponse.ID)

	err = s.runtimeStateStorage.Insert(
		internal.NewRuntimeState(*provisionerResponse.RuntimeID, operation.ID, requestInput.KymaConfig, nil),
	)
	if err != nil {
		log.Errorf("cannot insert runtimeState: %s", err)
		return operation, 10 * time.Second, nil
	}

	log.Infof("kyma upgrade process initiated successfully")
	// return repeat mode to start the initialization step which will now check the runtime status
	return operation, s.timeSchedule.Retry, nil
}

func (s *UpgradeKymaStep) createUpgradeKymaInput(operation internal.UpgradeKymaOperation) (gqlschema.UpgradeRuntimeInput, error) {
	var request gqlschema.UpgradeRuntimeInput

	request, err := operation.InputCreator.CreateUpgradeRuntimeInput()
	if err != nil {
		return request, errors.Wrap(err, "while building upgradeRuntimeInput for provisioner")
	}

	return request, nil
}
