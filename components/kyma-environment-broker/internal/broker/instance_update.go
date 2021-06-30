package broker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"

	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	"github.com/sirupsen/logrus"
)

type ContextUpdateHandler interface {
	Handle(instance *internal.Instance, newCtx internal.ERSContext) error
}

type UpdateEndpoint struct {
	config Config
	log    logrus.FieldLogger

	instanceStorage      storage.Instances
	contextUpdateHandler ContextUpdateHandler
	brokerURL            string
	processingEnabled    bool

	operationStorage storage.Operations

	updatingQueue *process.Queue
}

func NewUpdate(cfg Config,
	instanceStorage storage.Instances,
	operationStorage storage.Operations,
	ctxUpdateHandler ContextUpdateHandler,
	processingEnabled bool,
	queue *process.Queue,
	log logrus.FieldLogger,
) *UpdateEndpoint {
	return &UpdateEndpoint{
		config:               cfg,
		log:                  log.WithField("service", "UpdateEndpoint"),
		instanceStorage:      instanceStorage,
		operationStorage:     operationStorage,
		contextUpdateHandler: ctxUpdateHandler,
		processingEnabled:    processingEnabled,
		updatingQueue:        queue,
	}
}

// Update modifies an existing service instance
//  PATCH /v2/service_instances/{instance_id}
func (b *UpdateEndpoint) Update(_ context.Context, instanceID string, details domain.UpdateDetails, asyncAllowed bool) (domain.UpdateServiceSpec, error) {
	logger := b.log.WithField("instanceID", instanceID)
	logger.Infof("Updateing instanceID: %s", instanceID)
	logger.Infof("Updateing asyncAllowed: %v", asyncAllowed)
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

	var contextData map[string]interface{}
	err = json.Unmarshal(details.RawContext, &contextData)
	if err != nil {
		logger.Errorf("unable to unmarshal context: %s", err.Error())
		return domain.UpdateServiceSpec{}, errors.New("unable to unmarshal context")
	}
	logger.Infof("Context with keys:")
	for k, _ := range contextData {
		logger.Info(k)
	}

	lastProvisioningOperation, err := b.operationStorage.GetProvisioningOperationByInstanceID(instance.InstanceID)
	if err != nil {
		logger.Errorf("cannot fetch provisioning lastProvisioningOperation for instance with ID: %s : %s", instance.InstanceID, err.Error())
		return domain.UpdateServiceSpec{}, errors.New("unable to process the update")
	}

	if b.processingEnabled {
		instance, err := b.processContext(instance, details, lastProvisioningOperation, logger)
		if err != nil {
			return domain.UpdateServiceSpec{}, err
		}

		return b.processUpdateParameters(instance, details, lastProvisioningOperation, asyncAllowed, logger)
	}

	return domain.UpdateServiceSpec{
		IsAsync:       false,
		DashboardURL:  instance.DashboardURL,
		OperationData: "",
		Metadata: domain.InstanceMetadata{
			Labels: ResponseLabels(*lastProvisioningOperation, *instance, b.config.URL, b.config.EnableKubeconfigURLLabel),
		},
	}, nil
}

func (b *UpdateEndpoint) processUpdateParameters(instance *internal.Instance, details domain.UpdateDetails, lastProvisioningOperation *internal.ProvisioningOperation, asyncAllowed bool, logger logrus.FieldLogger) (domain.UpdateServiceSpec, error) {
	if len(details.RawParameters) == 0 {
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
	err := json.Unmarshal(details.RawParameters, &params)
	if err != nil {
		logger.Errorf("unable to unmarshal parameters: %s", err.Error())
		return domain.UpdateServiceSpec{}, errors.New("unable to unmarshal parametera")
	}
	logger.Debugf("Updating with params: %+v", params)

	operationID := uuid.New().String()
	logger = logger.WithField("operationID", operationID)

	logger.Debugf("creating update operation %v", params)
	operation := internal.NewUpdateOperation(operationID, instance, params)
	err = b.operationStorage.InsertUpdatingOperation(operation)
	if err != nil {
		return domain.UpdateServiceSpec{}, err
	}

	// update provisioning parameters in the instance
	if params.OIDC.IsProvided() {
		err = wait.Poll(500*time.Millisecond, 2*time.Second, func() (bool, error) {
			instance.Parameters.Parameters.OIDC = params.OIDC
			instance, err = b.instanceStorage.Update(*instance)
			if err != nil {
				logger.Warnf("unable to update instance with new parameters (%s), retrying", err.Error())
				return false, nil
			}
			return false, nil
		})
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

func (b *UpdateEndpoint) processContext(instance *internal.Instance, details domain.UpdateDetails, lastProvisioningOperation *internal.ProvisioningOperation, logger logrus.FieldLogger) (*internal.Instance, error) {
	var ersContext internal.ERSContext
	err := json.Unmarshal(details.RawContext, &ersContext)
	if err != nil {
		logger.Errorf("unable to decode context: %s", err.Error())
		return nil, errors.New("unable to unmarshal context")
	}
	logger.Infof("Global account ID: %s active: %s", instance.GlobalAccountID, ptr.BoolAsString(ersContext.Active))

	// todo: remove the code below when we are sure the ERSContext contains required values.
	// This code is done because the PATCH request contains only some of fields and that requests made the ERS context empty in the past.
	instance.Parameters.ErsContext = lastProvisioningOperation.ProvisioningParameters.ErsContext
	instance.Parameters.ErsContext.Active, err = b.exctractActiveValue(instance.InstanceID, *lastProvisioningOperation)
	if err != nil {
		return nil, errors.New("unable to process the update")
	}

	err = b.contextUpdateHandler.Handle(instance, ersContext)
	if err != nil {
		logger.Errorf("processing context updated failed: %s", err.Error())
		return nil, errors.New("unable to process the update")
	}

	//  copy the Active flag if set
	if ersContext.Active != nil {
		instance.Parameters.ErsContext.Active = ersContext.Active
	}

	newInstance, err := b.instanceStorage.Update(*instance)
	if err != nil {
		logger.Errorf("processing context updated failed: %s", err.Error())
		return nil, errors.New("unable to process the update")
	}
	return newInstance, nil
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
