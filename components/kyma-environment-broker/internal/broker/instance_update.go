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

type UpdateEndpoint struct {
	log logrus.FieldLogger

	instanceStorage storage.Instances
}

func NewUpdate(instanceStorage storage.Instances, log logrus.FieldLogger) *UpdateEndpoint {
	return &UpdateEndpoint{
		log:             log.WithField("service", "UpdateEndpoint"),
		instanceStorage: instanceStorage,
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

	return domain.UpdateServiceSpec{
		IsAsync:       false,
		DashboardURL:  instance.DashboardURL,
		OperationData: "",
	}, nil
}
