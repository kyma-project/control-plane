package update

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

// CheckStep checks if the SKR is updated
type CheckStep struct {
	provisionerClient   provisioner.Client
	operationManager    *process.UpdateOperationManager
	provisioningTimeout time.Duration
}

func NewCheckStep(os storage.Operations,
	provisionerClient provisioner.Client,
	provisioningTimeout time.Duration) *CheckStep {
	return &CheckStep{
		provisionerClient:   provisionerClient,
		operationManager:    process.NewUpdateOperationManager(os),
		provisioningTimeout: provisioningTimeout,
	}
}

var _ Step = (*CheckStep)(nil)

func (s *CheckStep) Name() string {
	return "Check_Runtime"
}

func (s *CheckStep) Run(operation internal.UpdatingOperation, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	if operation.RuntimeID == "" {
		log.Errorf("Runtime ID is empty")
		return s.operationManager.OperationFailed(operation, "Runtime ID is empty", log)
	}
	return s.checkRuntimeStatus(operation, log.WithField("runtimeID", operation.RuntimeID))
}

func (s *CheckStep) checkRuntimeStatus(operation internal.UpdatingOperation, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > s.provisioningTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", s.provisioningTimeout), log)
	}

	if operation.ProvisionerOperationID == "" {
		msg := "Operation dos not contain Provisioner Operation ID"
		log.Error(msg)
		return s.operationManager.OperationFailed(operation, msg, log)
	}

	status, err := s.provisionerClient.RuntimeOperationStatus(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.ProvisionerOperationID)
	if err != nil {
		log.Errorf("call to provisioner RuntimeOperationStatus failed: %s", err.Error())
		return operation, 1 * time.Minute, nil
	}
	log.Infof("call to provisioner returned %s status", status.State.String())

	var msg string
	if status.Message != nil {
		msg = *status.Message
	}

	switch status.State {
	case gqlschema.OperationStateSucceeded:
		return operation, 0, nil
	case gqlschema.OperationStateInProgress:
		return operation, time.Minute, nil
	case gqlschema.OperationStatePending:
		return operation, time.Minute, nil
	case gqlschema.OperationStateFailed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("provisioner client returns failed status: %s", msg), log)
	}

	return s.operationManager.OperationFailed(operation, fmt.Sprintf("unsupported provisioner client status: %s", status.State.String()), log)
}
