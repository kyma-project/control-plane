package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// label key used to send to director
	grafanaURLLabel = "operator_grafanaUrl"
)

//go:generate mockery -name=DirectorClient -output=automock -outpkg=automock -case=underscore

type DirectorClient interface {
	GetConsoleURL(accountID, runtimeID string) (string, error)
	SetLabel(accountID, runtimeID, key, value string) error
}

type KymaVersionConfigurator interface {
	ForGlobalAccount(string) (string, bool, error)
}

type SMClientFactory interface {
	ForCredentials(credentials *servicemanager.Credentials) servicemanager.Client
	ForCustomerCredentials(reqCredentials *servicemanager.Credentials, log logrus.FieldLogger) (servicemanager.Client, error)
	ProvideCredentials(reqCredentials *servicemanager.Credentials, log logrus.FieldLogger) (*servicemanager.Credentials, error)
}

type InitialisationStep struct {
	operationManager            *process.ProvisionOperationManager
	inputBuilder                input.CreatorForPlan
	operationTimeout            time.Duration
	provisioningTimeout         time.Duration
	runtimeVerConfigurator      RuntimeVersionConfiguratorForProvisioning
	serviceManagerClientFactory SMClientFactory
	instanceStorage             storage.Instances
}

func NewInitialisationStep(os storage.Operations, is storage.Instances,
	b input.CreatorForPlan,
	provisioningTimeout time.Duration,
	operationTimeout time.Duration,
	rvc RuntimeVersionConfiguratorForProvisioning,
	smcf SMClientFactory) *InitialisationStep {
	return &InitialisationStep{
		operationManager:            process.NewProvisionOperationManager(os),
		inputBuilder:                b,
		operationTimeout:            operationTimeout,
		provisioningTimeout:         provisioningTimeout,
		runtimeVerConfigurator:      rvc,
		serviceManagerClientFactory: smcf,
		instanceStorage:             is,
	}
}

func (s *InitialisationStep) Name() string {
	return "Provision_Initialization"
}

func (s *InitialisationStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	operation.SMClientFactory = s.serviceManagerClientFactory

	// configure the Kyma version to use
	err := s.configureKymaVersion(&operation, log)
	if err != nil {
		return s.operationManager.RetryOperation(operation, err.Error(), 5*time.Second, 5*time.Minute, log)
	}

	// create Provisioner InputCreator
	log.Infof("create provisioner input creator for %q plan ID", operation.ProvisioningParameters.PlanID)
	creator, err := s.inputBuilder.CreateProvisionInput(operation.ProvisioningParameters, operation.RuntimeVersion)

	switch {
	case err == nil:
		operation.InputCreator = creator

		err := s.updateInstance(operation.InstanceID, creator.Provider())
		if err != nil {
			return s.operationManager.RetryOperation(operation, err.Error(), 1*time.Second, 5*time.Second, log)
		}

		return operation, 0, nil
	case kebError.IsTemporaryError(err):
		log.Errorf("cannot create input creator at the moment for plan %s and version %s: %s", operation.ProvisioningParameters.PlanID, operation.ProvisioningParameters.Parameters.KymaVersion, err)
		return s.operationManager.RetryOperation(operation, err.Error(), 5*time.Second, 5*time.Minute, log)
	default:
		log.Errorf("cannot create input creator for plan %s: %s", operation.ProvisioningParameters.PlanID, err)
		return s.operationManager.OperationFailed(operation, "cannot create provisioning input creator", log)
	}
}

func (s *InitialisationStep) configureKymaVersion(operation *internal.ProvisioningOperation, log logrus.FieldLogger) error {
	if !operation.RuntimeVersion.IsEmpty() {
		return nil
	}
	version, err := s.runtimeVerConfigurator.ForProvisioning(*operation)
	if err != nil {
		return errors.Wrap(err, "while getting the runtime version")
	}

	var repeat time.Duration
	if *operation, repeat = s.operationManager.UpdateOperation(*operation, func(operation *internal.ProvisioningOperation) {
		operation.RuntimeVersion = *version
	}, log); repeat != 0 {
		return errors.New("unable to update operation with RuntimeVersion property")
	}
	return nil
}

func (s *InitialisationStep) updateInstance(id string, provider internal.CloudProvider) error {
	instance, err := s.instanceStorage.GetByID(id)
	if err != nil {
		return errors.Wrap(err, "while getting instance")
	}
	instance.Provider = provider
	_, err = s.instanceStorage.Update(*instance)
	if err != nil {
		return errors.Wrap(err, "while updating instance")
	}

	return nil
}
