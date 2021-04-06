package provisioning

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

// CheckRuntimeStep checks if the SKR is provisioned (expects proper label in the Director)
type CheckRuntimeStep struct {
	provisionerClient   provisioner.Client
	directorClient      DirectorClient
	operationManager    *process.ProvisionOperationManager
	provisioningTimeout time.Duration
}

func NewCheckRuntimeStep(os storage.Operations,
	provisionerClient provisioner.Client,
	directorClient DirectorClient,
	provisioningTimeout time.Duration) *CheckRuntimeStep {
	return &CheckRuntimeStep{
		provisionerClient:   provisionerClient,
		directorClient:      directorClient,
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
		return s.operationManager.OperationFailed(operation, "Runtime ID is empty", log)
	}
	return s.checkRuntimeStatus(operation, log.WithField("runtimeID", operation.RuntimeID))
}

func (s *CheckRuntimeStep) checkRuntimeStatus(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > s.provisioningTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", s.provisioningTimeout), log)
	}

	status, err := s.provisionerClient.RuntimeOperationStatus(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.ProvisionerOperationID)
	if err != nil {
		return operation, 1 * time.Minute, nil
	}
	log.Infof("call to provisioner returned %s status", status.State.String())

	var msg string
	if status.Message != nil {
		msg = *status.Message
	}

	switch status.State {
	case gqlschema.OperationStateSucceeded:
		repeat, err := s.handleDashboardURL(&operation, log)
		if repeat != 0 {
			return operation, repeat, nil
		}
		if err != nil {
			log.Errorf("cannot handle dashboard URL: %s", err)
			return s.operationManager.OperationFailed(operation, "cannot handle dashboard URL", log)
		}
		return operation, 0, nil
	case gqlschema.OperationStateInProgress:
		return operation, 2 * time.Minute, nil
	case gqlschema.OperationStatePending:
		return operation, 2 * time.Minute, nil
	case gqlschema.OperationStateFailed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("provisioner client returns failed status: %s", msg), log)
	}

	return s.operationManager.OperationFailed(operation, fmt.Sprintf("unsupported provisioner client status: %s", status.State.String()), log)
}

func (s *CheckRuntimeStep) handleDashboardURL(operation *internal.ProvisioningOperation, log logrus.FieldLogger) (time.Duration, error) {
	dashboardURL, err := s.directorClient.GetConsoleURL(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.RuntimeID)
	if kebError.IsTemporaryError(err) {
		log.Errorf("cannot get console URL from director client: %s", err)
		return 3 * time.Minute, nil
	}
	if err != nil {
		return 0, errors.Wrapf(err, "while getting URL from director")
	}

	if operation.DashboardURL != dashboardURL {
		return 0, errors.Errorf("dashboard URL from operation '%s' is not equal to dashboard URL from director '%s'", operation.DashboardURL, dashboardURL)
	}

	return 0, nil
}
