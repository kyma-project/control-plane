package upgrade_kyma

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

type MonitoringUpgradeStep struct {
	operationManager *process.UpgradeKymaOperationManager
	client           monitoring.Client
	monitoringConfig monitoring.Config
}

func NewMonitoringUpgradeStep(os storage.Operations, client monitoring.Client, monitoringConfig monitoring.Config) *MonitoringUpgradeStep {
	return &MonitoringUpgradeStep{
		operationManager: process.NewUpgradeKymaOperationManager(os),
		client:           client,
		monitoringConfig: monitoringConfig,
	}
}

func (s *MonitoringUpgradeStep) Name() string {
	return "Monitoring_Upgrade"
}

func (s *MonitoringUpgradeStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	releaseName := operation.InstanceDetails.ShootName
	isPresent, err := s.client.IsPresent(releaseName)
	if err != nil {
		return s.handleError(operation, err, "err while getting release", log)
	}
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
	}

	if !isPresent {
		log.Info("Release does not exist. Start installation.")
		params.Username = operation.InstanceID
		params.Password = monitoring.GeneratePassword(16)
		_, err = s.client.InstallRelease(params)
		if err != nil {
			return s.handleError(operation, err, "failed to install chart", log)
		}

		retry := time.Duration(0)
		operation, retry = s.operationManager.UpdateOperation(operation, func(operation *internal.UpgradeKymaOperation) {
			operation.Monitoring.Username = params.Username
			operation.Monitoring.Password = params.Password
		}, log)
		if retry > 0 {
			return operation, time.Second, nil
		}
	} else {
		log.Info("Release exists. Start upgrade.")
		params.Username = operation.Monitoring.Username
		params.Password = operation.Monitoring.Password
		_, err = s.client.UpgradeRelease(params)
		if err != nil {
			return s.handleError(operation, err, "helm release upgrade failed", log)
		}
	}

	log.Info("Override username and password")
	MonitoringOverrides := []*gqlschema.ConfigEntryInput{
		{
			Key:   "vmuser.username",
			Value: params.Username,
		},
		{
			Key:   "vmuser.password",
			Value: params.Password,
		},
	}
	operation.InputCreator.AppendOverrides(MonitoringComponentName, MonitoringOverrides)

	return operation, 0, nil
}

func (s *MonitoringUpgradeStep) handleError(operation internal.UpgradeKymaOperation, err error, msg string, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	switch {
	case kebError.IsTemporaryError(err):
		return s.operationManager.RetryOperation(operation, msg, 30*time.Second, 10*time.Minute, log)
	default:
		return s.operationManager.OperationFailed(operation, msg, log)
	}
}
