package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/monitoring"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
)

const (
	MonitoringComponentName = "rma"
)

type MonitoringIntegrationStep struct {
	operationManager *process.ProvisionOperationManager
	client           monitoring.Client
	monitoringConfig monitoring.Config
}

func NewMonitoringIntegrationStep(os storage.Operations, client monitoring.Client, monitoringConfig monitoring.Config) *MonitoringIntegrationStep {
	return &MonitoringIntegrationStep{
		operationManager: process.NewProvisionOperationManager(os),
		client:           client,
		monitoringConfig: monitoringConfig,
	}
}

func (s *MonitoringIntegrationStep) Name() string {
	return "Monitoring_Integration"
}

func (s *MonitoringIntegrationStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	releaseName := operation.InstanceDetails.ShootName
	isDeployed, err := s.client.IsDeployed(releaseName)
	isPresent, err2 := s.client.IsPresent(releaseName)
	if err != nil || err2 != nil {
		return s.handleError(operation, err, "err while getting release", log)
	}
	vmPassword := monitoring.GeneratePassword(16)
	planName, _ := broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]
	region := ""
	if operation.ProvisioningParameters.Parameters.Region != nil {
		region = *operation.ProvisioningParameters.Parameters.Region
	}
	params := monitoring.Parameters{
		ReleaseName:     releaseName,
		InstanceID:      operation.InstanceID,
		GlobalAccountID: operation.ProvisioningParameters.ErsContext.GlobalAccountID,
		SubaccountID:    operation.InstanceDetails.SubAccountID,
		ShootName:       operation.InstanceDetails.ShootName,
		PlanName:        planName,
		Region:          region,
		Username:        operation.InstanceID,
		Password:        vmPassword,
	}

	if !isDeployed {
		if !isPresent {
			log.Info("Release not found. Ready to install.")
			_, err = s.client.InstallRelease(params)
		} else {
			log.Info("Release found but not deployed. Ready to upgrade.")
			_, err = s.client.UpgradeRelease(params)
		}
		if err != nil {
			return s.handleError(operation, err, "failed to deploy chart", log)
		}
		retry := time.Duration(0)
		operation, retry = s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
			operation.Monitoring.Username = operation.InstanceID
			operation.Monitoring.Password = vmPassword
		}, log)
		if retry > 0 {
			return operation, time.Second, nil
		}
	} else {
		log.Info("Release already deployed.")
		vmPassword = operation.Monitoring.Password
	}

	log.Info("Override username and password")
	MonitoringOverrides := []*gqlschema.ConfigEntryInput{
		{
			Key:   "vmuser.username",
			Value: operation.InstanceID,
		},
		{
			Key:   "vmuser.password",
			Value: vmPassword,
		},
	}
	operation.InputCreator.AppendOverrides(MonitoringComponentName, MonitoringOverrides)

	return operation, 0, nil
}

func (s *MonitoringIntegrationStep) handleError(operation internal.ProvisioningOperation, err error, msg string, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	switch {
	case kebError.IsTemporaryError(err):
		return s.operationManager.RetryOperation(operation, msg, 10*time.Second, time.Minute*30, log)
	default:
		return s.operationManager.OperationFailed(operation, msg, log)
	}
}
