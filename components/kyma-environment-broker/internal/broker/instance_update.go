package broker

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/sirupsen/logrus"
)

type ContextUpdateHandler interface {
	Handle(instance *internal.Instance, newCtx internal.ERSContext) error
}

type UpdateEndpoint struct {
	log logrus.FieldLogger

	instanceStorage      storage.Instances
	contextUpdateHandler ContextUpdateHandler
	processingEnabled    bool

	operationStorage storage.Operations
}

func NewUpdate(instanceStorage storage.Instances, operationStorage storage.Operations, ctxUpdateHandler ContextUpdateHandler, processingEnabled bool, log logrus.FieldLogger) *UpdateEndpoint {
	return &UpdateEndpoint{
		log:                  log.WithField("service", "UpdateEndpoint"),
		instanceStorage:      instanceStorage,
		operationStorage:     operationStorage,
		contextUpdateHandler: ctxUpdateHandler,
		processingEnabled:    processingEnabled,
	}
}

// Update modifies an existing service instance
//  PATCH /v2/service_instances/{instance_id}
func (b *UpdateEndpoint) Update(ctx context.Context, instanceID string, details domain.UpdateDetails, asyncAllowed bool) (domain.UpdateServiceSpec, error) {
	logger := b.log.WithField("instanceID", instanceID)
	logger.Infof("Update instanceID: %s", instanceID)
	logger.Infof("Update asyncAllowed: %v", asyncAllowed)

	instance, err := b.instanceStorage.GetByID(instanceID)
	if err != nil {
		logger.Errorf("unable to get instance: %s", err.Error())
		return domain.UpdateServiceSpec{}, errors.New("unable to get instance")
	}
	logger.Infof("Plan ID/Name: %s/%s", instance.ServicePlanID, PlanIDsMapping[instance.ServicePlanID])

	var ersContext internal.ERSContext
	err = json.Unmarshal(details.RawContext, &ersContext)
	if err != nil {
		logger.Errorf("unable to decode context: %s", err.Error())
		return domain.UpdateServiceSpec{}, errors.New("unable to unmarshal context")
	}
	logger.Infof("Global account ID: %s active: %v", instance.GlobalAccountID, ersContext.Active)

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

	if b.processingEnabled {
		// todo: remove the code below when we are sure the ERSContext contains required values.
		// This code is done because the PATCH request contains only some of fields and that requests made the ERS context empty in the past.
		provOperation, err := b.operationStorage.GetProvisioningOperationByInstanceID(instanceID)
		if err != nil {
			logger.Errorf("processing context updated failed: %s", err.Error())
			return domain.UpdateServiceSpec{
				IsAsync:       false,
				DashboardURL:  instance.DashboardURL,
				OperationData: "",
			}, errors.New("unable to process the update")
		}
		instance.Parameters.ErsContext = provOperation.ProvisioningParameters.ErsContext

		err = b.contextUpdateHandler.Handle(instance, ersContext)
		if err != nil {
			logger.Errorf("processing context updated failed: %s", err.Error())
			return domain.UpdateServiceSpec{
				IsAsync:       false,
				DashboardURL:  instance.DashboardURL,
				OperationData: "",
			}, errors.New("unable to process the update")
		}

		//  copy only the Active flag
		instance.Parameters.ErsContext.Active = ersContext.Active

		_, err = b.instanceStorage.Update(*instance)
		if err != nil {
			logger.Errorf("processing context updated failed: %s", err.Error())
			return domain.UpdateServiceSpec{
				IsAsync:       false,
				DashboardURL:  instance.DashboardURL,
				OperationData: "",
			}, errors.New("unable to process the update")
		}
	}

	return domain.UpdateServiceSpec{
		IsAsync:       false,
		DashboardURL:  instance.DashboardURL,
		OperationData: "",
	}, nil
}
