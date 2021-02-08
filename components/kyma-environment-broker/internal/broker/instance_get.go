package broker

import (
	"context"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type GetInstanceEndpoint struct {
	instancesStorage storage.Instances

	log logrus.FieldLogger
}

func NewGetInstance(instancesStorage storage.Instances, log logrus.FieldLogger) *GetInstanceEndpoint {
	return &GetInstanceEndpoint{
		instancesStorage: instancesStorage,
		log:              log.WithField("service", "GetInstanceEndpoint"),
	}
}

// GetInstance fetches information about a service instance
//   GET /v2/service_instances/{instance_id}
func (b *GetInstanceEndpoint) GetInstance(ctx context.Context, instanceID string) (domain.GetInstanceDetailsSpec, error) {
	logger := b.log.WithField("instanceID", instanceID)
	logger.Infof("GetInstance called")

	inst, err := b.instancesStorage.GetByID(instanceID)
	if err != nil {
		return domain.GetInstanceDetailsSpec{}, errors.Wrapf(err, "while getting instance from storage")
	}

	spec := domain.GetInstanceDetailsSpec{
		ServiceID:    inst.ServiceID,
		PlanID:       inst.ServicePlanID,
		DashboardURL: inst.DashboardURL,
		Parameters:   inst.Parameters,
	}
	return spec, nil
}
