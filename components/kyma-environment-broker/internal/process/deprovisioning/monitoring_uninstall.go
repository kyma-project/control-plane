package deprovisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/monitoring"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type MonitoringUnistallStep struct {
	operationManager *process.DeprovisionOperationManager
	client           monitoring.Client
	monitoringConfig monitoring.Config
}

func NewMonitoringUnistallStep(os storage.Operations, client monitoring.Client, monitoringConfig monitoring.Config) *MonitoringUnistallStep {
	return &MonitoringUnistallStep{
		operationManager: process.NewDeprovisionOperationManager(os),
		client:           client,
		monitoringConfig: monitoringConfig,
	}
}

func (s *MonitoringUnistallStep) Name() string {
	return "Monitoring_Uninstall"
}

func (s *MonitoringUnistallStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	releaseName := operation.InstanceDetails.ShootName
	isPresent, err := s.client.IsPresent(releaseName)
	if err != nil {
		return s.handleError(operation, err, "failed to check release existence", log)
	}

	if isPresent {
		log.Info("Release found. Ready to unistall.")
		_, err = s.client.UninstallRelease(releaseName)
		if err != nil {
			return s.handleError(operation, err, "failed to uninstall release", log)
		}
	} else {
		log.Info("Release not found. Skip unistallation.")
	}
	return operation, 0, nil
}

func (s *MonitoringUnistallStep) handleError(operation internal.DeprovisioningOperation, err error, msg string, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	switch {
	case kebError.IsTemporaryError(err):
		return s.operationManager.RetryOperation(operation, msg, 10*time.Second, time.Minute*30, log)
	default:
		return s.operationManager.OperationFailed(operation, msg, log)
	}
}
