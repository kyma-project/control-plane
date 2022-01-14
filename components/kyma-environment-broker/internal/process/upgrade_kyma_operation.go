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

type UpgradeKymaOperationManager struct {
	storage storage.UpgradeKyma
}

func NewUpgradeKymaOperationManager(storage storage.Operations) *UpgradeKymaOperationManager {
	return &UpgradeKymaOperationManager{storage: storage}
}

// OperationSucceeded marks the operation as succeeded and only repeats it if there is a storage error
func (om *UpgradeKymaOperationManager) OperationSucceeded(operation internal.UpgradeKymaOperation, description string, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	updatedOperation, repeat, _ := om.update(operation, orchestration.Succeeded, description, log)
	// repeat in case of storage error
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	return updatedOperation, 0, nil
}

// OperationFailed marks the operation as failed and only repeats it if there is a storage error
func (om *UpgradeKymaOperationManager) OperationFailed(operation internal.UpgradeKymaOperation, description string, err error, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	updatedOperation, repeat, _ := om.update(operation, orchestration.Failed, description, log)
	// repeat in case of storage error
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	return updatedOperation, 0, errors.Wrap(err, description)
}

// OperationSucceeded marks the operation as succeeded and only repeats it if there is a storage error
func (om *UpgradeKymaOperationManager) OperationCanceled(operation internal.UpgradeKymaOperation, description string, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	updatedOperation, repeat, _ := om.update(operation, orchestration.Canceled, description, log)
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	return updatedOperation, 0, nil
}

// RetryOperation retries an operation for at maxTime in retryInterval steps and fails the operation if retrying failed
func (om *UpgradeKymaOperationManager) RetryOperation(operation internal.UpgradeKymaOperation, errorMessage string, err error, retryInterval time.Duration, maxTime time.Duration, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
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
func (om *UpgradeKymaOperationManager) UpdateOperation(operation internal.UpgradeKymaOperation, update func(operation *internal.UpgradeKymaOperation), log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	update(&operation)
	updatedOperation, err := om.storage.UpdateUpgradeKymaOperation(operation)
	switch {
	case dberr.IsConflict(err):
		{
			op, err := om.storage.GetUpgradeKymaOperationByID(operation.Operation.ID)
			if err != nil {
				log.Errorf("while getting operation: %v", err)
				return operation, 1 * time.Minute, err
			}
			update(op)
			updatedOperation, err = om.storage.UpdateUpgradeKymaOperation(*op)
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

// Deprecated: SimpleUpdateOperation updates a given operation without handling conflicts. Should be used when operation's data mutations are not clear
func (om *UpgradeKymaOperationManager) SimpleUpdateOperation(operation internal.UpgradeKymaOperation) (internal.UpgradeKymaOperation, time.Duration) {
	updatedOperation, err := om.storage.UpdateUpgradeKymaOperation(operation)
	if err != nil {
		logrus.WithField("orchestrationID", operation.OrchestrationID).
			WithField("instanceID", operation.InstanceID).
			Errorf("Update provisioning operation failed: %s", err.Error())
		return operation, 1 * time.Minute
	}
	return *updatedOperation, 0
}

// RetryOperationWithoutFail retries an operation for at maxTime in retryInterval steps and omits the operation if retrying failed
func (om *UpgradeKymaOperationManager) RetryOperationWithoutFail(operation internal.UpgradeKymaOperation, description string, retryInterval, maxTime time.Duration, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
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

func (om *UpgradeKymaOperationManager) update(operation internal.UpgradeKymaOperation, state domain.LastOperationState, description string, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	return om.UpdateOperation(operation, func(operation *internal.UpgradeKymaOperation) {
		operation.State = state
		operation.Description = description
	}, log)
}
