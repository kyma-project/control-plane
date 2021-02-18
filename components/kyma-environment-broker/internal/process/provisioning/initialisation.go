package provisioning

import (
	"fmt"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

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

type InitialisationStep struct {
	operationManager            *process.ProvisionOperationManager
	instanceStorage             storage.Instances
	provisionerClient           provisioner.Client
	directorClient              DirectorClient
	inputBuilder                input.CreatorForPlan
	externalEvalCreator         *ExternalEvalCreator
	internalEvalUpdater         *InternalEvalUpdater
	iasType                     *IASType
	operationTimeout            time.Duration
	provisioningTimeout         time.Duration
	runtimeVerConfigurator      RuntimeVersionConfiguratorForProvisioning
	serviceManagerClientFactory *servicemanager.ClientFactory
}

func NewInitialisationStep(os storage.Operations,
	is storage.Instances,
	pc provisioner.Client,
	dc DirectorClient,
	b input.CreatorForPlan,
	avsExternalEvalCreator *ExternalEvalCreator,
	avsInternalEvalUpdater *InternalEvalUpdater,
	iasType *IASType,
	provisioningTimeout time.Duration,
	operationTimeout time.Duration,
	rvc RuntimeVersionConfiguratorForProvisioning,
	smcf *servicemanager.ClientFactory) *InitialisationStep {
	return &InitialisationStep{
		operationManager:            process.NewProvisionOperationManager(os),
		instanceStorage:             is,
		provisionerClient:           pc,
		directorClient:              dc,
		inputBuilder:                b,
		externalEvalCreator:         avsExternalEvalCreator,
		internalEvalUpdater:         avsInternalEvalUpdater,
		iasType:                     iasType,
		operationTimeout:            operationTimeout,
		provisioningTimeout:         provisioningTimeout,
		runtimeVerConfigurator:      rvc,
		serviceManagerClientFactory: smcf,
	}
}

func (s *InitialisationStep) Name() string {
	return "Provision_Initialization"
}

func (s *InitialisationStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if time.Since(operation.CreatedAt) > s.operationTimeout {
		log.Infof("operation has reached the time limit: operation was created at: %s", operation.CreatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", s.operationTimeout))
	}
	operation.SMClientFactory = s.serviceManagerClientFactory

	inst, err := s.instanceStorage.GetByID(operation.InstanceID)
	switch {
	case err == nil:
		if inst.RuntimeID == "" {
			log.Info("runtimeID not exist, initialize runtime input request")
			return s.initializeRuntimeInputRequest(operation, log)
		}
		log.Info("runtimeID exist, check instance status")
		return s.checkRuntimeStatus(operation, log.WithField("runtimeID", inst.RuntimeID))
	case dberr.IsNotFound(err):
		log.Info("instance not exist")
		return s.operationManager.OperationFailed(operation, "instance was not created")
	default:
		log.Errorf("unable to get instance from storage: %s", err)
		return operation, 1 * time.Second, nil
	}
}

func (s *InitialisationStep) initializeRuntimeInputRequest(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	err := s.configureKymaVersion(&operation)
	if err != nil {
		return s.operationManager.RetryOperation(operation, err.Error(), 5*time.Second, 5*time.Minute, log)
	}

	log.Infof("create provisioner input creator for %q plan ID", operation.ProvisioningParameters.PlanID)
	creator, err := s.inputBuilder.CreateProvisionInput(operation.ProvisioningParameters, operation.RuntimeVersion)
	switch {
	case err == nil:
		operation.InputCreator = creator
		return operation, 0, nil
	case kebError.IsTemporaryError(err):
		log.Errorf("cannot create input creator at the moment for plan %s and version %s: %s", operation.ProvisioningParameters.PlanID, operation.ProvisioningParameters.Parameters.KymaVersion, err)
		return s.operationManager.RetryOperation(operation, err.Error(), 5*time.Second, 5*time.Minute, log)
	default:
		log.Errorf("cannot create input creator for plan %s: %s", operation.ProvisioningParameters.PlanID, err)
		return s.operationManager.OperationFailed(operation, "cannot create provisioning input creator")
	}
}

func (s *InitialisationStep) configureKymaVersion(operation *internal.ProvisioningOperation) error {
	if !operation.RuntimeVersion.IsEmpty() {
		return nil
	}
	version, err := s.runtimeVerConfigurator.ForProvisioning(*operation)
	if err != nil {
		return errors.Wrap(err, "while getting the runtime version")
	}

	operation.RuntimeVersion = *version

	var repeat time.Duration
	if *operation, repeat = s.operationManager.UpdateOperation(*operation); repeat != 0 {
		return errors.New("unable to update operation with RuntimeVersion property")
	}
	return nil
}

func (s *InitialisationStep) checkRuntimeStatus(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > s.provisioningTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", s.provisioningTimeout))
	}

	instance, err := s.instanceStorage.GetByID(operation.InstanceID)
	if err != nil {
		return operation, 10 * time.Second, nil
	}

	status, err := s.provisionerClient.RuntimeOperationStatus(instance.GlobalAccountID, operation.ProvisionerOperationID)
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
		repeat, err := s.handleDashboardURL(instance, log)
		if repeat != 0 {
			return operation, repeat, nil
		}
		if err != nil {
			log.Errorf("cannot handle dashboard URL: %s", err)
			return s.operationManager.OperationFailed(operation, "cannot handle dashboard URL")
		}
		return s.launchPostActions(operation, instance, log, msg)
	case gqlschema.OperationStateInProgress:
		return operation, 2 * time.Minute, nil
	case gqlschema.OperationStatePending:
		return operation, 2 * time.Minute, nil
	case gqlschema.OperationStateFailed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("provisioner client returns failed status: %s", msg))
	}

	return s.operationManager.OperationFailed(operation, fmt.Sprintf("unsupported provisioner client status: %s", status.State.String()))
}

func (s *InitialisationStep) handleDashboardURL(instance *internal.Instance, log logrus.FieldLogger) (time.Duration, error) {
	dashboardURL, err := s.directorClient.GetConsoleURL(instance.GlobalAccountID, instance.RuntimeID)
	if kebError.IsTemporaryError(err) {
		log.Errorf("cannot get console URL from director client: %s", err)
		return 3 * time.Minute, nil
	}
	if err != nil {
		return 0, errors.Wrapf(err, "while getting URL from director")
	}

	if instance.DashboardURL != dashboardURL {
		return 0, errors.Errorf("dashboard URL from instance %s is not equal to dashboard URL from director %s", instance.DashboardURL, dashboardURL)
	}

	return 0, nil
}

func (s *InitialisationStep) launchPostActions(operation internal.ProvisioningOperation, instance *internal.Instance, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	// action #1
	operation, repeat, err := s.createExternalEval(operation, instance, log)
	if err != nil || repeat != 0 {
		if err != nil {
			log.Errorf("while creating external Evaluation: %s", err)
			return operation, repeat, err
		}
		return operation, repeat, nil
	}

	// action #2
	tags, operation, repeat, err := s.createTagsForRuntime(operation, instance)
	if err != nil || repeat != 0 {
		log.Errorf("while creating Tags for Evaluation: %s", err)
		return operation, repeat, nil
	}
	operation, repeat, err = s.internalEvalUpdater.AddTagsToEval(tags, operation, "", log)
	if err != nil || repeat != 0 {
		log.Errorf("while adding Tags to Evaluation: %s", err)
		return operation, repeat, nil
	}

	// action #3
	repeat, err = s.iasType.ConfigureType(operation, instance.DashboardURL, log)
	if err != nil || repeat != 0 {
		return operation, repeat, nil
	}
	if !s.iasType.Disabled() {
		grafanaPath := strings.Replace(instance.DashboardURL, "console.", "grafana.", 1)
		err = s.directorClient.SetLabel(instance.GlobalAccountID, instance.RuntimeID, grafanaURLLabel, grafanaPath)
		if err != nil {
			log.Errorf("Cannot set labels in director: %s", err)
		} else {
			log.Infof("Label %s:%s set correctly", grafanaURLLabel, instance.DashboardURL)
		}
	}

	return s.operationManager.OperationSucceeded(operation, msg)
}

func (s *InitialisationStep) createExternalEval(operation internal.ProvisioningOperation, instance *internal.Instance, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if operation.ProvisioningParameters.PlanID == broker.TrialPlanID {
		log.Info("skipping AVS external evaluation creation for trial plan")
		return operation, 0, nil
	}
	log.Infof("creating external evaluation for instance %", instance.InstanceID)
	operation, repeat, err := s.externalEvalCreator.createEval(operation, instance.DashboardURL, log)
	if err != nil || repeat != 0 {
		return operation, repeat, err
	}
	return operation, 0, nil
}

func (s *InitialisationStep) createTagsForRuntime(operation internal.ProvisioningOperation, instance *internal.Instance) ([]*avs.Tag, internal.ProvisioningOperation, time.Duration, error) {

	status, err := s.provisionerClient.RuntimeStatus(instance.GlobalAccountID, operation.RuntimeID)
	if err != nil {
		return []*avs.Tag{}, operation, 1 * time.Minute, err
	}

	result := []*avs.Tag{
		{
			Content:    ptr.ToString(status.RuntimeConfiguration.ClusterConfig.Name),
			TagClassId: s.internalEvalUpdater.avsConfig.GardenerShootNameTagClassId,
		},
		{
			Content:    ptr.ToString(status.RuntimeConfiguration.ClusterConfig.Seed),
			TagClassId: s.internalEvalUpdater.avsConfig.GardenerSeedNameTagClassId,
		},
		{
			Content:    ptr.ToString(status.RuntimeConfiguration.ClusterConfig.Region),
			TagClassId: s.internalEvalUpdater.avsConfig.RegionTagClassId,
		},
	}

	return result, operation, 0 * time.Second, nil
}
