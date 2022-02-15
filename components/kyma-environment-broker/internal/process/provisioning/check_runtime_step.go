package provisioning

import (
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

// CheckRuntimeStep checks if the SKR is provisioned
type CheckRuntimeStep struct {
	provisionerClient   provisioner.Client
	operationManager    *process.ProvisionOperationManager
	provisioningTimeout time.Duration
}

func NewCheckRuntimeStep(os storage.Operations,
	provisionerClient provisioner.Client,
	provisioningTimeout time.Duration) *CheckRuntimeStep {
	return &CheckRuntimeStep{
		provisionerClient:   provisionerClient,
		operationManager:    process.NewProvisionOperationManager(os),
		provisioningTimeout: provisioningTimeout,
	}
}

var _ Step = (*CheckRuntimeStep)(nil)

func (s *CheckRuntimeStep) Name() string {
	return "Check_Runtime"
}

func (s *CheckRuntimeStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if operation.RuntimeID == "" {
		log.Errorf("Runtime ID is empty")
		return s.operationManager.OperationFailed(operation, "Runtime ID is empty", errors.New(""), log)
	}
	return s.checkRuntimeStatus(operation, log.WithField("runtimeID", operation.RuntimeID))
}

func (s *CheckRuntimeStep) checkRuntimeStatus(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	emptyErr := errors.New("")

	if time.Since(operation.UpdatedAt) > s.provisioningTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", s.provisioningTimeout), emptyErr, log)
	}

	if operation.ProvisionerOperationID == "" {
		msg := "Operation dos not contain Provisioner Operation ID"
		log.Error(msg)
		return s.operationManager.OperationFailed(operation, msg, emptyErr, log)
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
		return operation, 2 * time.Minute, nil
	case gqlschema.OperationStatePending:
		return operation, 2 * time.Minute, nil
	case gqlschema.OperationStateFailed:
		lastErr := provisioner.OperationStatusLastError(status.LastError)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("provisioner client returns failed status: %s", msg), lastErr, log)
	}

	lastErr := provisioner.OperationStatusLastError(status.LastError)
	return s.operationManager.OperationFailed(operation, fmt.Sprintf("unsupported provisioner client status: %s", status.State.String()), lastErr, log)
}
