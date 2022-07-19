package process

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/pkg/errors"

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

// OperationSucceeded marks the operation as succeeded and only repeats it if there is a storage error
func (om *OperationManager) OperationSucceeded(operation internal.Operation, description string, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	op, repeat, _ := om.update(operation, domain.Succeeded, description, log)
	// repeat in case of storage error
	if repeat != 0 {
		return op, repeat, nil
	}

	return op, 0, nil
}

// OperationFailed marks the operation as failed and only repeats it if there is a storage error
func (om *OperationManager) OperationFailed(operation internal.Operation, description string, err error, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	op, repeat, _ := om.update(operation, domain.Failed, description, log)
	// repeat in case of storage error
	if repeat != 0 {
		return op, repeat, nil
	}

	var retErr error
	if err == nil {
		// no exact err passed in
		retErr = errors.New(description)
	} else {
		// keep the original err object for error categorizer
		retErr = errors.Wrap(err, description)
	}

	return op, 0, retErr
}

// OperationSucceeded marks the operation as succeeded and only repeats it if there is a storage error
func (om *OperationManager) OperationCanceled(operation internal.Operation, description string, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	op, repeat, _ := om.update(operation, orchestration.Canceled, description, log)
	if repeat != 0 {
		return op, repeat, nil
	}

	return op, 0, nil
}

// RetryOperation retries an operation for at maxTime in retryInterval steps and fails the operation if retrying failed
func (om *OperationManager) RetryOperation(operation internal.Operation, errorMessage string, err error, retryInterval time.Duration, maxTime time.Duration, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	since := time.Since(operation.UpdatedAt)

	log.Infof("Retry Operation was triggered with message: %s", errorMessage)
	log.Infof("Retrying for %s in %s steps", maxTime.String(), retryInterval.String())
	if since < maxTime {
		return operation, retryInterval, nil
	}
	log.Errorf("Aborting after %s of failing retries", maxTime.String())
	return om.OperationFailed(operation, errorMessage, err, log)
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

// RetryOperationWithoutFail retries an operation for at maxTime in retryInterval steps and omits the operation if retrying failed
func (om *OperationManager) RetryOperationWithoutFail(operation internal.Operation, description string, retryInterval, maxTime time.Duration, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	since := time.Since(operation.UpdatedAt)

	log.Infof("Retry Operation was triggered with message: %s", description)
	log.Infof("Retrying for %s in %s steps", maxTime.String(), retryInterval.String())
	if since < maxTime {
		return operation, retryInterval, nil
	}
	// update description to track failed steps
	op, repeat, _ := om.update(operation, domain.InProgress, description, log)
	if repeat != 0 {
		return op, repeat, nil
	}

	log.Errorf("Omitting after %s of failing retries", maxTime.String())
	return op, 0, nil
}

func (om *OperationManager) update(operation internal.Operation, state domain.LastOperationState, description string, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	return om.UpdateOperation(operation, func(operation *internal.Operation) {
		operation.State = state
		operation.Description = description
	}, log)
}
