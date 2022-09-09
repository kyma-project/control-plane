package deprovisioning

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

type CheckRuntimeRemovalStep struct {
	operationManager  *process.OperationManager
	provisionerClient provisioner.Client
}

var _ process.Step = &CheckRuntimeRemovalStep{}

func NewCheckRuntimeRemovalStep(operations storage.Operations, provisionerClient provisioner.Client) *CheckRuntimeRemovalStep {
	return &CheckRuntimeRemovalStep{
		operationManager:  process.NewOperationManager(operations),
		provisionerClient: provisionerClient,
	}
}

func (s *CheckRuntimeRemovalStep) Name() string {
	return "Check_Runtime_Removal"
}

func (s *CheckRuntimeRemovalStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > CheckStatusTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", CheckStatusTimeout), nil, log)
	}

	status, err := s.provisionerClient.RuntimeOperationStatus(operation.GlobalAccountID, operation.ProvisionerOperationID)
	if err != nil {
		log.Errorf("call to provisioner RuntimeOperationStatus failed: %s", err.Error())
		return operation, 1 * time.Minute, nil
	}
	var msg string
	if status.Message != nil {
		msg = *status.Message
	}
	log.Infof("call to provisioner returned %s status: %s", status.State.String(), msg)

	switch status.State {
	case gqlschema.OperationStateSucceeded:
		return operation, 0, nil
	case gqlschema.OperationStateInProgress:
		return operation, 30 * time.Second, nil
	case gqlschema.OperationStatePending:
		return operation, 30 * time.Second, nil
	case gqlschema.OperationStateFailed:
		lastErr := provisioner.OperationStatusLastError(status.LastError)
		return s.operationManager.OperationFailed(operation, "provisioner client returns failed status", lastErr, log)
	}

	lastErr := provisioner.OperationStatusLastError(status.LastError)
	return s.operationManager.OperationFailed(operation, fmt.Sprintf("unsupported provisioner client status: %s", status.State.String()), lastErr, log)
}
