package process

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
)

type OperationManager struct {
	storage storage.Operations
}

func NewOperationManager(storage storage.Operations) *OperationManager {
	return &OperationManager{storage: storage}
}

// OperationSucceeded marks the operation as succeeded and returns status of the operation's update
func (om *OperationManager) OperationSucceeded(operation internal.Operation, description string, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	return om.update(operation, domain.Succeeded, description, log)
}

// OperationFailed marks the operation as failed and returns status of the operation's update
func (om *OperationManager) OperationFailed(operation internal.Operation, description string, err error, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	return om.update(operation, domain.Failed, description, log)
}

// OperationCanceled marks the operation as canceled and returns status of the operation's update
func (om *OperationManager) OperationCanceled(operation internal.Operation, description string, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	return om.update(operation, orchestration.Canceled, description, log)
}

// RetryOperation checks if operation should be retried or if it's the status should be marked as failed
func (om *OperationManager) RetryOperation(operation internal.Operation, errorMessage string, err error, retryInterval time.Duration, maxTime time.Duration, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	log.Infof("Retry Operation was triggered with message: %s", errorMessage)
	log.Infof("Retrying for %s in %s steps", maxTime.String(), retryInterval.String())
	if time.Since(operation.UpdatedAt) < maxTime {
		return operation, retryInterval, nil
	}
	log.Errorf("Aborting after %s of failing retries", maxTime.String())
	op, retry, err := om.OperationFailed(operation, errorMessage, err, log)
	if err == nil {
		err = fmt.Errorf("Too many retries")
	} else {
		err = fmt.Errorf("Failed to set status for operation after too many retries: %v", err)
	}
	return op, retry, err
}

// RetryOperationWithoutFail checks if operation should be retried or updates the status to InProgress, but omits setting the operation to failed if maxTime is reached
func (om *OperationManager) RetryOperationWithoutFail(operation internal.Operation, description string, retryInterval, maxTime time.Duration, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	log.Infof("Retry Operation was triggered with message: %s", description)
	log.Infof("Retrying for %s in %s steps", maxTime.String(), retryInterval.String())
	if time.Since(operation.UpdatedAt) < maxTime {
		return operation, retryInterval, nil
	}
	// update description to track failed steps
	op, repeat, err := om.update(operation, domain.InProgress, description, log)
	if repeat != 0 {
		return op, repeat, err
	}

	log.Errorf("Omitting after %s of failing retries", maxTime.String())
	return op, 0, nil
}

// RetryOperationOnce retries the operation once and fails the operation when call second time
func (om *OperationManager) RetryOperationOnce(operation internal.Operation, errorMessage string, err error, wait time.Duration, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	return om.RetryOperation(operation, errorMessage, err, wait, wait+1, log)
}

// UpdateOperation updates a given operation and handles conflict situation
func (om *OperationManager) UpdateOperation(operation internal.Operation, update func(operation *internal.Operation), log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	update(&operation)
	op, err := om.storage.UpdateOperation(operation)
	switch {
	case dberr.IsConflict(err):
		{
			op, err := om.storage.GetOperationByID(operation.ID)
			if err != nil {
				log.Errorf("while getting operation: %v", err)
				return operation, 1 * time.Minute, err
			}
			update(op)
			op, err = om.storage.UpdateOperation(*op)
			if err != nil {
				log.Errorf("while updating operation after conflict: %v", err)
				return operation, 1 * time.Minute, err
			}
		}
	case err != nil:
		log.Errorf("while updating operation: %v", err)
		return operation, 1 * time.Minute, err
	}
	return *op, 0, nil
}

func (om *OperationManager) update(operation internal.Operation, state domain.LastOperationState, description string, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	return om.UpdateOperation(operation, func(operation *internal.Operation) {
		operation.State = state
		operation.Description = description
	}, log)
}
