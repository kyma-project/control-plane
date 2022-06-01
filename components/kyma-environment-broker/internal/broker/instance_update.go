package broker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/dashboard"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
)

type ContextUpdateHandler interface {
	Handle(instance *internal.Instance, newCtx internal.ERSContext) (bool, error)
}

type UpdateEndpoint struct {
	config Config
	log    logrus.FieldLogger

	instanceStorage           storage.Instances
	runtimeStates             storage.RuntimeStates
	contextUpdateHandler      ContextUpdateHandler
	brokerURL                 string
	processingEnabled         bool
	subAccountMovementEnabled bool

	operationStorage storage.Operations

	updatingQueue *process.Queue

	planDefaults PlanDefaults

	dashboardConfig dashboard.Config
}

func NewUpdate(cfg Config,
	instanceStorage storage.Instances,
	runtimeStates storage.RuntimeStates,
	operationStorage storage.Operations,
	ctxUpdateHandler ContextUpdateHandler,
	processingEnabled bool,
	subAccountMovementEnabled bool,
	queue *process.Queue,
	planDefaults PlanDefaults,
	log logrus.FieldLogger,
	dashboardConfig dashboard.Config,
) *UpdateEndpoint {
	return &UpdateEndpoint{
		config:                    cfg,
		log:                       log.WithField("service", "UpdateEndpoint"),
		instanceStorage:           instanceStorage,
		runtimeStates:             runtimeStates,
		operationStorage:          operationStorage,
		contextUpdateHandler:      ctxUpdateHandler,
		processingEnabled:         processingEnabled,
		subAccountMovementEnabled: subAccountMovementEnabled,
		updatingQueue:             queue,
		planDefaults:              planDefaults,
		dashboardConfig:           dashboardConfig,
	}
}

// Update modifies an existing service instance
//  PATCH /v2/service_instances/{instance_id}
func (b *UpdateEndpoint) Update(_ context.Context, instanceID string, details domain.UpdateDetails, asyncAllowed bool) (domain.UpdateServiceSpec, error) {
	logger := b.log.WithField("instanceID", instanceID)
	logger.Infof("Updating instanceID: %s", instanceID)
	logger.Infof("Updating asyncAllowed: %v", asyncAllowed)
	logger.Infof("Parameters: '%s'", string(details.RawParameters))

	instance, err := b.instanceStorage.GetByID(instanceID)
	if err != nil && dberr.IsNotFound(err) {
		logger.Errorf("unable to get instance: %s", err.Error())
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusNotFound, fmt.Sprintf("could not execute update for instanceID %s", instanceID))
	} else if err != nil {
		logger.Errorf("unable to get instance: %s", err.Error())
		return domain.UpdateServiceSpec{}, errors.New("unable to get instance")
	}
	logger.Infof("Plan ID/Name: %s/%s", instance.ServicePlanID, PlanNamesMapping[instance.ServicePlanID])

	var ersContext internal.ERSContext
	err = json.Unmarshal(details.RawContext, &ersContext)
	if err != nil {
		logger.Errorf("unable to decode context: %s", err.Error())
		return domain.UpdateServiceSpec{}, errors.New("unable to unmarshal context")
	}
	logger.Infof("Global account ID: %s active: %s", instance.GlobalAccountID, ptr.BoolAsString(ersContext.Active))
	logger.Infof("Migration triggered: %v", ersContext.IsMigration)
	logger.Infof("Received context: %s", marshallRawContext(hideSensitiveDataFromRawContext(details.RawContext)))

	lastProvisioningOperation, err := b.operationStorage.GetProvisioningOperationByInstanceID(instance.InstanceID)
	if err != nil {
		logger.Errorf("cannot fetch provisioning lastProvisioningOperation for instance with ID: %s : %s", instance.InstanceID, err.Error())
		return domain.UpdateServiceSpec{}, errors.New("unable to process the update")
	}
	if lastProvisioningOperation.State == domain.Failed {
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(errors.New("Unable to process an update of a failed instance"), http.StatusUnprocessableEntity, "")
	}

	lastDeprovisioningOperation, err := b.operationStorage.GetDeprovisioningOperationByInstanceID(instance.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		logger.Errorf("cannot fetch deprovisioning for instance with ID: %s : %s", instance.InstanceID, err.Error())
		return domain.UpdateServiceSpec{}, errors.New("unable to process the update")
	}
	if err == nil {
		if !lastDeprovisioningOperation.Temporary {
			// it is not a suspension, but real deprovisioning
			logger.Warnf("Cannot process update, the instance has started deprovisioning process (operationID=%s)", lastDeprovisioningOperation.Operation.ID)
			return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(errors.New("Unable to process an update of a deprovisioned instance"), http.StatusUnprocessableEntity, "")
		}
	}

	dashboardURL := instance.DashboardURL
	if b.dashboardConfig.Enabled && b.dashboardConfig.LandscapeURL != "" {
		dashboardURL = fmt.Sprintf("%s/?kubeconfigID=%s", b.dashboardConfig.LandscapeURL, instanceID)
		instance.DashboardURL = dashboardURL
	}

	if b.processingEnabled {
		instance, suspendStatusChange, err := b.processContext(instance, details, lastProvisioningOperation, logger)
		if err != nil {
			return domain.UpdateServiceSpec{}, err
		}

		// NOTE: KEB currently can't process update parameters in one call along with context update
		// this block makes it that KEB ignores any parameters updates if context update changed suspension state
		if !suspendStatusChange {
			return b.processUpdateParameters(instance, details, lastProvisioningOperation, asyncAllowed, ersContext, logger)
		}
	}

	return domain.UpdateServiceSpec{
		IsAsync:       false,
		DashboardURL:  dashboardURL,
		OperationData: "",
		Metadata: domain.InstanceMetadata{
			Labels: ResponseLabels(*lastProvisioningOperation, *instance, b.config.URL, b.config.EnableKubeconfigURLLabel),
		},
	}, nil
}

func shouldUpdate(instance *internal.Instance, details domain.UpdateDetails, ersContext internal.ERSContext) bool {
	if len(details.RawParameters) != 0 {
		return true
	}
	return instance.InstanceDetails.SCMigrationTriggered || ersContext.ERSUpdate()
}

func (b *UpdateEndpoint) processUpdateParameters(instance *internal.Instance, details domain.UpdateDetails, lastProvisioningOperation *internal.ProvisioningOperation, asyncAllowed bool, ersContext internal.ERSContext, logger logrus.FieldLogger) (domain.UpdateServiceSpec, error) {
	if !shouldUpdate(instance, details, ersContext) {
		logger.Debugf("Parameters not provided, skipping processing update parameters")
		return domain.UpdateServiceSpec{
			IsAsync:       false,
			DashboardURL:  instance.DashboardURL,
			OperationData: "",
			Metadata: domain.InstanceMetadata{
				Labels: ResponseLabels(*lastProvisioningOperation, *instance, b.config.URL, b.config.EnableKubeconfigURLLabel),
			},
		}, nil
	}
	// asyncAllowed needed, see https://github.com/openservicebrokerapi/servicebroker/blob/v2.16/spec.md#updating-a-service-instance
	if !asyncAllowed {
		return domain.UpdateServiceSpec{}, apiresponses.ErrAsyncRequired
	}
	var params internal.UpdatingParametersDTO
	if len(details.RawParameters) != 0 {
		err := json.Unmarshal(details.RawParameters, &params)
		if err != nil {
			logger.Errorf("unable to unmarshal parameters: %s", err.Error())
			return domain.UpdateServiceSpec{}, errors.New("unable to unmarshal parameters")
		}
		logger.Debugf("Updating with params: %+v", params)
	}

	if params.OIDC.IsProvided() {
		if err := params.OIDC.Validate(); err != nil {
			logger.Errorf("invalid OIDC parameters: %s", err.Error())
			return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusUnprocessableEntity, err.Error())
		}
	}

	operationID := uuid.New().String()
	logger = logger.WithField("operationID", operationID)

	logger.Debugf("creating update operation %v", params)
	operation := internal.NewUpdateOperation(operationID, instance, params)
	operation.InstanceDetails.SCMigrationTriggered = ersContext.IsMigration
	planID := instance.Parameters.PlanID
	if len(details.PlanID) != 0 {
		planID = details.PlanID
	}
	defaults, err := b.planDefaults(planID, instance.Provider, &instance.Provider)
	if err != nil {
		logger.Errorf("unable to obtain plan defaults: %s", err.Error())
		return domain.UpdateServiceSpec{}, errors.New("unable to obtain plan defaults")
	}
	var autoscalerMin, autoscalerMax int
	if defaults.GardenerConfig != nil {
		p := defaults.GardenerConfig
		autoscalerMin, autoscalerMax = p.AutoScalerMin, p.AutoScalerMax
	}
	if err := operation.ProvisioningParameters.Parameters.AutoScalerParameters.Validate(autoscalerMin, autoscalerMax); err != nil {
		logger.Errorf("invalid autoscaler parameters: %s", err.Error())
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusUnprocessableEntity, err.Error())
	}
	err = b.operationStorage.InsertUpdatingOperation(operation)
	if err != nil {
		return domain.UpdateServiceSpec{}, err
	}

	var updateStorage []string
	if params.OIDC.IsProvided() {
		instance.Parameters.Parameters.OIDC = params.OIDC
		updateStorage = append(updateStorage, "OIDC")
	}

	if len(params.RuntimeAdministrators) != 0 {
		newAdministrators := make([]string, 0, len(params.RuntimeAdministrators))
		newAdministrators = append(newAdministrators, params.RuntimeAdministrators...)
		instance.Parameters.Parameters.RuntimeAdministrators = newAdministrators
		updateStorage = append(updateStorage, "Runtime Administrators")
	}

	if params.UpdateAutoScaler(&instance.Parameters.Parameters) {
		updateStorage = append(updateStorage, "Auto Scaler parameters")
	}
	if len(updateStorage) > 0 {
		if err := wait.Poll(500*time.Millisecond, 2*time.Second, func() (bool, error) {
			instance, err = b.instanceStorage.Update(*instance)
			if err != nil {
				params := strings.Join(updateStorage, ", ")
				logger.Warnf("unable to update instance with new %v (%s), retrying", params, err.Error())
				return false, nil
			}
			return true, nil
		}); err != nil {
			response := apiresponses.NewFailureResponse(fmt.Errorf("Update operation failed"), http.StatusInternalServerError, err.Error())
			return domain.UpdateServiceSpec{}, response
		}
	}
	logger.Debugf("Adding update operation to the processing queue")
	b.updatingQueue.Add(operationID)

	return domain.UpdateServiceSpec{
		IsAsync:       true,
		DashboardURL:  instance.DashboardURL,
		OperationData: operation.ID,
		Metadata: domain.InstanceMetadata{
			Labels: ResponseLabels(*lastProvisioningOperation, *instance, b.config.URL, b.config.EnableKubeconfigURLLabel),
		},
	}, nil
}

func (b *UpdateEndpoint) processContext(instance *internal.Instance, details domain.UpdateDetails, lastProvisioningOperation *internal.ProvisioningOperation, logger logrus.FieldLogger) (*internal.Instance, bool, error) {
	var ersContext internal.ERSContext
	err := json.Unmarshal(details.RawContext, &ersContext)
	if err != nil {
		logger.Errorf("unable to decode context: %s", err.Error())
		return nil, false, errors.New("unable to unmarshal context")
	}
	logger.Infof("Global account ID: %s active: %s", instance.GlobalAccountID, ptr.BoolAsString(ersContext.Active))

	// todo: remove the code below when we are sure the ERSContext contains required values.
	// This code is done because the PATCH request contains only some of fields and that requests made the ERS context empty in the past.
	existingSMOperatorCredentials := instance.Parameters.ErsContext.SMOperatorCredentials
	instance.Parameters.ErsContext = lastProvisioningOperation.ProvisioningParameters.ErsContext
	// but do not change existing SM operator credentials
	instance.Parameters.ErsContext.SMOperatorCredentials = existingSMOperatorCredentials
	instance.Parameters.ErsContext.Active, err = b.exctractActiveValue(instance.InstanceID, *lastProvisioningOperation)
	if err != nil {
		return nil, false, errors.New("unable to process the update")
	}
	if ersContext.ServiceManager != nil {
		instance.Parameters.ErsContext.ServiceManager = ersContext.ServiceManager
	}
	if ersContext.SMOperatorCredentials != nil {
		instance.Parameters.ErsContext.SMOperatorCredentials = ersContext.SMOperatorCredentials
	}
	if ersContext.IsMigration {
		instance.Parameters.ErsContext.IsMigration = ersContext.IsMigration
		instance.InstanceDetails.SCMigrationTriggered = true
	}
	if ersContext.CommercialModel != nil {
		instance.Parameters.ErsContext.CommercialModel = ersContext.CommercialModel
	}
	if ersContext.LicenseType != nil {
		instance.Parameters.ErsContext.LicenseType = ersContext.LicenseType
	}
	if ersContext.Origin != nil {
		instance.Parameters.ErsContext.Origin = ersContext.Origin
	}
	if ersContext.Platform != nil {
		instance.Parameters.ErsContext.Platform = ersContext.Platform
	}
	if ersContext.Region != nil {
		instance.Parameters.ErsContext.Region = ersContext.Region
	}

	changed, err := b.contextUpdateHandler.Handle(instance, ersContext)
	if err != nil {
		logger.Errorf("processing context updated failed: %s", err.Error())
		return nil, changed, errors.New("unable to process the update")
	}

	//  copy the Active flag if set
	if ersContext.Active != nil {
		instance.Parameters.ErsContext.Active = ersContext.Active
	}

	if b.subAccountMovementEnabled {
		if instance.GlobalAccountID != ersContext.GlobalAccountID && ersContext.GlobalAccountID != "" {
			if instance.SubscriptionGlobalAccountID == "" {
				instance.SubscriptionGlobalAccountID = instance.GlobalAccountID
			}
			instance.GlobalAccountID = ersContext.GlobalAccountID
		}
	}

	newInstance, err := b.instanceStorage.Update(*instance)
	if err != nil {
		logger.Errorf("processing context updated failed: %s", err.Error())
		return nil, changed, errors.New("unable to process the update")
	}

	return newInstance, changed, nil
}

func (b *UpdateEndpoint) exctractActiveValue(id string, provisioning internal.ProvisioningOperation) (*bool, error) {
	deprovisioning, dErr := b.operationStorage.GetDeprovisioningOperationByInstanceID(id)
	if dErr != nil && !dberr.IsNotFound(dErr) {
		b.log.Errorf("Unable to get deprovisioning operation for the instance %s to check the active flag: %s", id, dErr.Error())
		return nil, dErr
	}
	// there was no any deprovisioning in the past (any suspension)
	if deprovisioning == nil {
		return ptr.Bool(true), nil
	}

	return ptr.Bool(deprovisioning.CreatedAt.Before(provisioning.CreatedAt)), nil
}

func (b *UpdateEndpoint) isKyma2(instance *internal.Instance) (bool, string, error) {
	s, err := b.runtimeStates.GetLatestWithKymaVersionByRuntimeID(instance.RuntimeID)
	if err != nil {
		return false, "", err
	}
	kv := s.GetKymaVersion()
	return internal.DetermineMajorVersion(kv) == 2, kv, nil
}
