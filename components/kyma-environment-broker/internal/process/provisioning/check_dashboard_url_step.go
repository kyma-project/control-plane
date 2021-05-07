package provisioning

import (
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

// CheckDashboardURLStep checks if the SKR is provisioned by checking proper label in the Director
type CheckDashboardURLStep struct {
	directorClient      DirectorClient
	operationManager    *process.ProvisionOperationManager
	provisioningTimeout time.Duration
}

func NewCheckDashboardURLStep(os storage.Operations,
	directorClient DirectorClient,
	provisioningTimeout time.Duration) *CheckDashboardURLStep {
	return &CheckDashboardURLStep{
		directorClient:      directorClient,
		operationManager:    process.NewProvisionOperationManager(os),
		provisioningTimeout: provisioningTimeout,
	}
}

var _ Step = (*CheckDashboardURLStep)(nil)

func (s *CheckDashboardURLStep) Name() string {
	return "Check_Dashboard_URL"
}

func (s *CheckDashboardURLStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	repeat, err := s.handleDashboardURL(&operation, log)
	if repeat != 0 {
		return operation, repeat, nil
	}
	if err != nil {
		log.Errorf("cannot handle dashboard URL: %s", err)
		return s.operationManager.OperationFailed(operation, "cannot handle dashboard URL", log)
	}
	return operation, 0, nil
}

func (s *CheckDashboardURLStep) handleDashboardURL(operation *internal.ProvisioningOperation, log logrus.FieldLogger) (time.Duration, error) {
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
