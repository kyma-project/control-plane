package broker

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"

	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/pivotal-cf/brokerapi/v7/domain/apiresponses"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type LastOperationEndpoint struct {
	operationStorage storage.Operations
	instancesStorage storage.Instances

	log logrus.FieldLogger
}

func NewLastOperation(os storage.Operations, is storage.Instances, log logrus.FieldLogger) *LastOperationEndpoint {
	return &LastOperationEndpoint{
		operationStorage: os,
		instancesStorage: is,
		log:              log.WithField("service", "LastOperationEndpoint"),
	}
}

// LastOperation fetches last operation state for a service instance
//   GET /v2/service_instances/{instance_id}/last_operation
func (b *LastOperationEndpoint) LastOperation(ctx context.Context, instanceID string, details domain.PollDetails) (domain.LastOperation, error) {
	logger := b.log.WithField("instanceID", instanceID).WithField("operationID", details.OperationData)

	if details.OperationData == "" {
		_, err := b.instancesStorage.GetByID(instanceID)
		switch {
		case err == nil:
			err = errors.New("operation data must be provided for asynchronous operations")
			return domain.LastOperation{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, err.Error())
		case dberr.IsNotFound(err):
			return domain.LastOperation{}, apiresponses.NewFailureResponse(errors.Errorf("instance does not exist"), http.StatusGone, fmt.Sprintf("instance with ID %s is not found in DB", instanceID))
		default:
			logger.Errorf("unable to get instance from a storage: %s", err)
			return domain.LastOperation{}, apiresponses.NewFailureResponse(errors.Errorf("unable to get instance from the storage"), http.StatusInternalServerError, fmt.Sprintf("could not get instance from DB, instanceID %s", instanceID))
		}
	}

	operation, err := b.operationStorage.GetOperationByID(details.OperationData)
	if err != nil {
		logger.Errorf("cannot get operation from storage: %s", err)
		return domain.LastOperation{}, errors.Wrapf(err, "while getting operation from storage")
	}

	if operation.InstanceID != instanceID {
		err := errors.Errorf("operation does not exist")
		return domain.LastOperation{}, apiresponses.NewFailureResponseBuilder(err, http.StatusBadRequest, err.Error())
	}

	return domain.LastOperation{
		State:       operation.State,
		Description: operation.Description,
	}, nil
}
