package broker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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
}

func NewUpdate(cfg Config,
	instanceStorage storage.Instances,
	operationStorage storage.Operations,
	ctxUpdateHandler ContextUpdateHandler,
	processingEnabled bool,
	log logrus.FieldLogger,
) *UpdateEndpoint {
	return &UpdateEndpoint{
		config:               cfg,
		log:                  log.WithField("service", "UpdateEndpoint"),
		instanceStorage:      instanceStorage,
		operationStorage:     operationStorage,
		contextUpdateHandler: ctxUpdateHandler,
		processingEnabled:    processingEnabled,
	}
}

// Update modifies an existing service instance
//  PATCH /v2/service_instances/{instance_id}
func (b *UpdateEndpoint) Update(_ context.Context, instanceID string, details domain.UpdateDetails, asyncAllowed bool) (domain.UpdateServiceSpec, error) {
	logger := b.log.WithField("instanceID", instanceID)
	logger.Infof("Update instanceID: %s", instanceID)
	logger.Infof("Update asyncAllowed: %v", asyncAllowed)

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

	operation, err := b.operationStorage.GetProvisioningOperationByInstanceID(instanceID)
	if err != nil {
		logger.Errorf("cannot fetch provisioning operation for instance with ID: %s : %s", instanceID, err.Error())
		return domain.UpdateServiceSpec{}, errors.New("unable to process the update")
	}

	if b.processingEnabled {
		// todo: remove the code below when we are sure the ERSContext contains required values.
		// This code is done because the PATCH request contains only some of fields and that requests made the ERS context empty in the past.
		instance.Parameters.ErsContext = operation.ProvisioningParameters.ErsContext
		instance.Parameters.ErsContext.Active, err = b.exctractActiveValue(instance.InstanceID, *operation)
		if err != nil {
			return domain.UpdateServiceSpec{}, errors.New("unable to process the update")
		}

		err = b.contextUpdateHandler.Handle(instance, ersContext)
		if err != nil {
			logger.Errorf("processing context updated failed: %s", err.Error())
			return domain.UpdateServiceSpec{}, errors.New("unable to process the update")
		}

		//  copy the Active flag if set
		if ersContext.Active != nil {
			instance.Parameters.ErsContext.Active = ersContext.Active
		}

		_, err = b.instanceStorage.Update(*instance)
		if err != nil {
			logger.Errorf("processing context updated failed: %s", err.Error())
			return domain.UpdateServiceSpec{}, errors.New("unable to process the update")
		}
	}

	return domain.UpdateServiceSpec{
		IsAsync:       false,
		DashboardURL:  instance.DashboardURL,
		OperationData: "",
		Metadata: domain.InstanceMetadata{
			Labels: ResponseLabels(*operation, *instance, b.config.URL, b.config.EnableKubeconfigURLLabel),
		},
	}, nil
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
