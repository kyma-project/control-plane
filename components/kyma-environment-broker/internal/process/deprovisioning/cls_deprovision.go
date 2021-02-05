package deprovisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type ClsDeprovisionStep struct {
	config           *cls.Config
	operationManager *process.DeprovisionOperationManager
}

func NewClsDeprovisionStep(config *cls.Config, os storage.Operations) *ClsDeprovisionStep {
	return &ClsDeprovisionStep{
		config:           config,
		operationManager: process.NewDeprovisionOperationManager(os),
	}
}

func (s *ClsDeprovisionStep) Name() string {
	return "CLS_Deprovision"
}

func (s *ClsDeprovisionStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (
	internal.DeprovisioningOperation, time.Duration, error) {
	if !operation.Cls.Instance.Provisioned {
		log.Infof("CLS deprovisioning step was already successfully finished")
		return operation, 0, nil
	}

	skrRegion := operation.ProvisioningParameters.Parameters.Region
	smCli, err := cls.ServiceManagerClient(operation.SMClientFactory, s.config.ServiceManager, skrRegion)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manage client"))
	}

	log.Infof("deprovisioning for CLS instance: %s started", operation.Cls.Instance.InstanceID)
	_, err = smCli.Deprovision(operation.Cls.Instance.InstanceKey(), false)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("Deprovision() call failed"))
	}
	log.Infof("deprovisioning for CLS instance: %s finished", operation.Ems.Instance.InstanceID)

	operation.Ems.Instance.InstanceID = ""
	operation.Ems.Instance.Provisioned = false
	return s.operationManager.UpdateOperation(operation)
}

func (s *ClsDeprovisionStep) handleError(operation internal.DeprovisioningOperation, err error, log logrus.FieldLogger,
	msg string) (internal.DeprovisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg)
}
