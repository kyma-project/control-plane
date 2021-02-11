package provisioning

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

//go:generate mockery --name=ClsInstanceProvider --output=automock --outpkg=automock --case=underscore
type ClsInstanceProvider interface {
	ProvisionIfNoneExists(smClient servicemanager.Client, request *cls.ProvisionRequest) (*cls.ProvisionResult, error)
}

type clsProvisioningStep struct {
	config           *cls.Config
	instanceProvider ClsInstanceProvider
	operationManager *process.ProvisionOperationManager
}

func NewClsProvisioningStep(config *cls.Config, ip ClsInstanceProvider, repo storage.Operations) *clsProvisioningStep {
	return &clsProvisioningStep{
		config:           config,
		operationManager: process.NewProvisionOperationManager(repo),
		instanceProvider: ip,
	}
}

func (s *clsProvisioningStep) Name() string {
	return "CLS_Provision"
}

func (s *clsProvisioningStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if operation.Cls.Instance.ProvisioningTriggered {
		return operation, 0, nil
	}

	skrRegion := operation.ProvisioningParameters.Parameters.Region
	smClient, err := cls.ServiceManagerClient(operation.SMClientFactory, s.config.ServiceManager, skrRegion)
	if smClient == nil {
		return operation, time.Second, errors.Wrapf(err, "service manager client could not be instantiated")
	}

	globalAccountID := operation.ProvisioningParameters.ErsContext.GlobalAccountID
	skrInstanceID := operation.InstanceID
	result, err := s.instanceProvider.ProvisionIfNoneExists(smClient, &cls.ProvisionRequest{
		GlobalAccountID: globalAccountID,
		SKRInstanceID:   skrInstanceID,
		BrokerID:        operation.Cls.Instance.BrokerID,
		ServiceID:       operation.Cls.Instance.ServiceID,
		PlanID:          operation.Cls.Instance.PlanID,
	})
	operation.Cls.Instance.InstanceID = result.InstanceID
	operation.Cls.Instance.ProvisioningTriggered = result.ProvisioningTriggered

	if err != nil {
		failureReason := fmt.Sprintf("Unable to create instance for GlobalAccountID: %s", globalAccountID)
		log.Errorf("%s: %s", failureReason, err)
		return s.operationManager.OperationFailed(operation, failureReason)
	}

	_, repeat := s.operationManager.UpdateOperation(operation)
	if repeat != 0 {
		log.Errorf("Unable to update operation: %s", err)
		return operation, time.Second, nil
	}

	return operation, 0, nil
}
