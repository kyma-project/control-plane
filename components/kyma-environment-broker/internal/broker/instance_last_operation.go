package broker

import (
	"context"
	"fmt"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"net/http"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type LastOperationEndpoint struct {
	operationStorage storage.Operations

	log logrus.FieldLogger
}

func NewLastOperation(os storage.Operations, log logrus.FieldLogger) *LastOperationEndpoint {
	return &LastOperationEndpoint{
		operationStorage: os,
		log:              log.WithField("service", "LastOperationEndpoint"),
	}
}

// LastOperation fetches last operation state for a service instance
//   GET /v2/service_instances/{instance_id}/last_operation
func (b *LastOperationEndpoint) LastOperation(ctx context.Context, instanceID string, details domain.PollDetails) (domain.LastOperation, error) {
	logger := b.log.WithField("instanceID", instanceID).WithField("operationID", details.OperationData)

	if details.OperationData == "" {
		lastOp, err := b.operationStorage.GetLastOperation(instanceID)
		if err != nil {
			logger.Errorf("cannot get operation from storage: %s", err)
			statusCode := http.StatusInternalServerError
			if dberr.IsNotFound(err){
				statusCode = http.StatusNotFound
			}
			return domain.LastOperation{}, apiresponses.NewFailureResponse(err, statusCode,
				fmt.Sprintf("while getting last operation from storage"))
		}
		return domain.LastOperation{
			State:       lastOp.State,
			Description: lastOp.Description,
		}, nil
	}

	operation, err := b.operationStorage.GetOperationByID(details.OperationData)
	if err != nil {
		logger.Errorf("cannot get operation from storage: %s", err)
		statusCode := http.StatusInternalServerError
		if dberr.IsNotFound(err){
			statusCode = http.StatusNotFound
		}
		return domain.LastOperation{}, apiresponses.NewFailureResponse(err, statusCode,
			fmt.Sprintf("while getting operation from storage"))
		//return domain.LastOperation{}, errors.Wrapf(err, "while getting operation from storage")
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
