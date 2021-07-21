package process

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type UpdateOperationManager struct {
	storage storage.Updating
}

func NewUpdateOperationManager(storage storage.Operations) *UpdateOperationManager {
	return &UpdateOperationManager{storage: storage}
}

// OperationSucceeded marks the operation as succeeded and only repeats it if there is a storage error
func (om *UpdateOperationManager) OperationSucceeded(operation internal.UpdatingOperation, description string, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	updatedOperation, repeat := om.update(operation, orchestration.Succeeded, description, log)
	// repeat in case of storage error
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	return updatedOperation, 0, nil
}

// OperationFailed marks the operation as failed and only repeats it if there is a storage error
func (om *UpdateOperationManager) OperationFailed(operation internal.UpdatingOperation, description string, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	updatedOperation, repeat := om.update(operation, orchestration.Failed, description, log)
	// repeat in case of storage error
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	return updatedOperation, 0, errors.New(description)
}

// RetryOperation retries an operation for at maxTime in retryInterval steps and fails the operation if retrying failed
func (om *UpdateOperationManager) RetryOperation(operation internal.UpdatingOperation, errorMessage string, retryInterval time.Duration, maxTime time.Duration, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	since := time.Since(operation.UpdatedAt)

	log.Infof("Retry Operation was triggered with message: %s", errorMessage)
	log.Infof("Retrying for %s in %s steps", maxTime.String(), retryInterval.String())
	if since < maxTime {
		return operation, retryInterval, nil
	}
	log.Errorf("Aborting after %s of failing retries", maxTime.String())
	return om.OperationFailed(operation, errorMessage, log)
}

// UpdateOperation updates a given operation
func (om *UpdateOperationManager) UpdateOperation(operation internal.UpdatingOperation, update func(operation *internal.UpdatingOperation), log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration) {
	update(&operation)
	updatedOperation, err := om.storage.UpdateUpdatingOperation(operation)
	switch {
	case dberr.IsConflict(err):
		{
			op, err := om.storage.GetUpdatingOperationByID(operation.Operation.ID)
			if err != nil {
				log.Errorf("while getting operation: %v", err)
				return operation, 1 * time.Minute
			}
			update(op)
			updatedOperation, err = om.storage.UpdateUpdatingOperation(*op)
			if err != nil {
				log.Errorf("while updating operation after conflict: %v", err)
				return operation, 1 * time.Minute
			}
		}
	case err != nil:
		log.Errorf("while updating operation: %v", err)
		return operation, 1 * time.Minute
	}
	return *updatedOperation, 0
}

// Deprecated: SimpleUpdateOperation updates a given operation without handling conflicts. Should be used when operation's data mutations are not clear
func (om *UpdateOperationManager) SimpleUpdateOperation(operation internal.UpdatingOperation) (internal.UpdatingOperation, time.Duration) {
	updatedOperation, err := om.storage.UpdateUpdatingOperation(operation)
	if err != nil {
		logrus.WithField("orchestrationID", operation.OrchestrationID).
			WithField("instanceID", operation.InstanceID).
			Errorf("Update upgradeCluster operation failed: %s", err.Error())
		return operation, 1 * time.Minute
	}
	return *updatedOperation, 0
}

// RetryOperationWithoutFail retries an operation for at maxTime in retryInterval steps and omits the operation if retrying failed
func (om *UpdateOperationManager) RetryOperationWithoutFail(operation internal.UpdatingOperation, description string, retryInterval, maxTime time.Duration, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	since := time.Since(operation.UpdatedAt)

	log.Infof("Retry Operation was triggered with message: %s", description)
	log.Infof("Retrying for %s in %s steps", maxTime.String(), retryInterval.String())
	if since < maxTime {
		return operation, retryInterval, nil
	}
	// update description to track failed steps
	updatedOperation, repeat := om.update(operation, domain.InProgress, description, log)
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	log.Errorf("Omitting after %s of failing retries", maxTime.String())
	return updatedOperation, 0, nil
}

func (om *UpdateOperationManager) update(operation internal.UpdatingOperation, state domain.LastOperationState, description string, log logrus.FieldLogger) (internal.UpdatingOperation, time.Duration) {
	return om.UpdateOperation(operation, func(operation *internal.UpdatingOperation) {
		operation.State = state
		operation.Description = description
	}, log)
}
