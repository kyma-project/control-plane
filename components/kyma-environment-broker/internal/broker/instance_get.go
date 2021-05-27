package broker

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"

	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/pivotal-cf/brokerapi/v7/domain/apiresponses"
	"github.com/sirupsen/logrus"
)

type GetInstanceEndpoint struct {
	instancesStorage  storage.Instances
	operationsStorage storage.Provisioning

	log logrus.FieldLogger
}

func NewGetInstance(instancesStorage storage.Instances, operationsStorage storage.Provisioning, log logrus.FieldLogger) *GetInstanceEndpoint {
	return &GetInstanceEndpoint{
		instancesStorage:  instancesStorage,
		operationsStorage: operationsStorage,
		log:               log.WithField("service", "GetInstanceEndpoint"),
	}
}

// GetInstance fetches information about a service instance
//   GET /v2/service_instances/{instance_id}
func (b *GetInstanceEndpoint) GetInstance(ctx context.Context, instanceID string) (domain.GetInstanceDetailsSpec, error) {
	logger := b.log.WithField("instanceID", instanceID)
	logger.Infof("GetInstance called")

	inst, err := b.instancesStorage.GetByID(instanceID)
	if err != nil {
		statusCode := http.StatusNotFound
		if !dberr.IsNotFound(err) {
			statusCode = http.StatusInternalServerError
		}
		return domain.GetInstanceDetailsSpec{}, apiresponses.NewFailureResponse(err, statusCode, fmt.Sprintf("failed to get instanceID %s", instanceID))
	}

	// check if provisioning still in progress
	if op, err := b.operationsStorage.GetProvisioningOperationByInstanceID(instanceID); err != nil {
		return domain.GetInstanceDetailsSpec{}, apiresponses.NewFailureResponse(err, http.StatusNotFound, fmt.Sprintf("failed to get operation for instanceID %s", instanceID))
	} else if op.State == domain.InProgress || op.State == domain.Failed {
		err = fmt.Errorf("provisioning of instanceID %s %s", instanceID, op.State)
		return domain.GetInstanceDetailsSpec{}, apiresponses.NewFailureResponse(err, http.StatusNotFound, err.Error())
	}

	spec := domain.GetInstanceDetailsSpec{
		ServiceID:    inst.ServiceID,
		PlanID:       inst.ServicePlanID,
		DashboardURL: inst.DashboardURL,
		Parameters:   inst.Parameters,
	}
	return spec, nil
}
