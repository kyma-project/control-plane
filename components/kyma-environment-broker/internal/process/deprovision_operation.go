package process

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pkg/errors"
)

type DeprovisionOperationManager struct {
	storage storage.Operations
}

func NewDeprovisionOperationManager(storage storage.Operations) *DeprovisionOperationManager {
	return &DeprovisionOperationManager{
		storage: storage,
	}
}

// OperationSucceeded marks the operation as succeeded and only repeats it if there is a storage error
func (om *DeprovisionOperationManager) OperationSucceeded(operation internal.DeprovisioningOperation, description string, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	updatedOperation, repeat, _ := om.update(operation, domain.Succeeded, description, log)
	// repeat in case of storage error
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	return updatedOperation, 0, nil
}

// OperationFailed marks the operation as failed and only repeats it if there is a storage error
func (om *DeprovisionOperationManager) OperationFailed(operation internal.DeprovisioningOperation, description string, err error, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	updatedOperation, repeat, _ := om.update(operation, domain.Failed, description, log)
	// repeat in case of storage error
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	return updatedOperation, 0, errors.Wrap(err, description)
}

// UpdateOperation updates a given operation and handles conflict situation
func (om *DeprovisionOperationManager) UpdateOperation(operation internal.DeprovisioningOperation, overwrite func(operation *internal.DeprovisioningOperation), log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	overwrite(&operation)
	updatedOperation, err := om.storage.UpdateDeprovisioningOperation(operation)
	switch {
	case dberr.IsConflict(err):
		{
			op, err := om.storage.GetDeprovisioningOperationByID(operation.ID)
			if err != nil {
				log.Errorf("while getting operation: %v", err)
				return operation, 1 * time.Minute, err
			}
			overwrite(op)
			updatedOperation, err = om.storage.UpdateDeprovisioningOperation(*op)
			if err != nil {
				log.Errorf("while updating operation after conflict: %v", err)
				return operation, 1 * time.Minute, err
			}
		}
	case err != nil:
		log.Errorf("while updating operation: %v", err)
		return operation, 1 * time.Minute, err
	}
	return *updatedOperation, 0, nil
}

// InsertOperation stores operation in database
func (om *DeprovisionOperationManager) InsertOperation(operation internal.DeprovisioningOperation) (internal.DeprovisioningOperation, time.Duration, error) {
	err := om.storage.InsertDeprovisioningOperation(operation)
	if err != nil {
		return operation, 1 * time.Minute, nil
	}
	return operation, 0, nil
}

// RetryOperationOnce retries the operation once and fails the operation when call second time
func (om *DeprovisionOperationManager) RetryOperationOnce(operation internal.DeprovisioningOperation, errorMessage string, err error, wait time.Duration, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	return om.RetryOperation(operation, errorMessage, err, wait, wait+1, log)
}

// RetryOperation retries an operation for at maxTime in retryInterval steps and fails the operation if retrying failed
func (om *DeprovisionOperationManager) RetryOperation(operation internal.DeprovisioningOperation, errorMessage string, err error, retryInterval time.Duration, maxTime time.Duration, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	since := time.Since(operation.UpdatedAt)

	log.Infof("Retrying for %s in %s steps, error: %s", maxTime.String(), retryInterval.String(), errorMessage)
	if since < maxTime {
		return operation, retryInterval, nil
	}
	log.Errorf("Aborting after %s of failing retries", maxTime.String())
	return om.OperationFailed(operation, errorMessage, err, log)
}

// RetryOperationWithoutFail retries an operation for at maxTime in retryInterval steps and omits the operation if retrying failed
func (om *DeprovisionOperationManager) RetryOperationWithoutFail(operation internal.DeprovisioningOperation, description string, retryInterval, maxTime time.Duration, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	since := time.Since(operation.UpdatedAt)

	log.Infof("Retry Operation was triggered with message: %s", description)
	log.Infof("Retrying for %s in %s steps", maxTime.String(), retryInterval.String())
	if since < maxTime {
		return operation, retryInterval, nil
	}
	// update description to track failed steps
	updatedOperation, repeat, _ := om.update(operation, domain.InProgress, description, log)
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	log.Errorf("Omitting after %s of failing retries", maxTime.String())
	return updatedOperation, 0, nil
}

func (om *DeprovisionOperationManager) update(operation internal.DeprovisioningOperation, state domain.LastOperationState, description string, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	return om.UpdateOperation(operation, func(operation *internal.DeprovisioningOperation) {
		operation.State = state
		operation.Description = fmt.Sprintf("%s : %s", operation.Description, description)
	}, log)
}
