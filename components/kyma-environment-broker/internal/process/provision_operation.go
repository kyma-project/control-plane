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

type ProvisionOperationManager struct {
	storage storage.Provisioning
}

func NewProvisionOperationManager(storage storage.Operations) *ProvisionOperationManager {
	return &ProvisionOperationManager{storage: storage}
}

// OperationSucceeded marks the operation as succeeded and only repeats it if there is a storage error
func (om *ProvisionOperationManager) OperationSucceeded(operation internal.ProvisioningOperation, description string, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	updatedOperation, repeat := om.update(operation, domain.Succeeded, description, log)
	// repeat in case of storage error
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	return updatedOperation, 0, nil
}

// OperationFailed marks the operation as failed and only repeats it if there is a storage error
func (om *ProvisionOperationManager) OperationFailed(operation internal.ProvisioningOperation, description string, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	updatedOperation, repeat := om.update(operation, domain.Failed, description, log)
	// repeat in case of storage error
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	return updatedOperation, 0, errors.New(description)
}

// UpdateOperation updates a given operation and handles conflict situation
func (om *ProvisionOperationManager) UpdateOperation(operation internal.ProvisioningOperation, update func(operation *internal.ProvisioningOperation), log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration) {
	update(&operation)
	updatedOperation, err := om.storage.UpdateProvisioningOperation(operation)
	switch {
	case dberr.IsConflict(err):
		{
			op, err := om.storage.GetProvisioningOperationByID(operation.ID)
			if err != nil {
				log.Errorf("while getting operation: %v", err)
				return operation, 1 * time.Minute
			}
			update(op)
			updatedOperation, err = om.storage.UpdateProvisioningOperation(*op)
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
func (om *ProvisionOperationManager) SimpleUpdateOperation(operation internal.ProvisioningOperation) (internal.ProvisioningOperation, time.Duration) {
	updatedOperation, err := om.storage.UpdateProvisioningOperation(operation)
	if err != nil {
		logrus.WithField("operation", operation.ID).
			WithField("instanceID", operation.InstanceID).
			Errorf("Update provisioning operation failed: %s", err.Error())
		return operation, 1 * time.Minute
	}
	return *updatedOperation, 0
}

// RetryOperationOnce retries the operation once and fails the operation when call second time
func (om *ProvisionOperationManager) RetryOperationOnce(operation internal.ProvisioningOperation, errorMessage string, wait time.Duration, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	return om.RetryOperation(operation, errorMessage, wait, wait+1, log)
}

// RetryOperation retries an operation for at maxTime in retryInterval steps and fails the operation if retrying failed
func (om *ProvisionOperationManager) RetryOperation(operation internal.ProvisioningOperation, errorMessage string, retryInterval time.Duration, maxTime time.Duration, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	since := time.Since(operation.UpdatedAt)

	log.Infof("Retry Operation was triggered with message: %s", errorMessage)
	log.Infof("Retrying for %s in %s steps", maxTime.String(), retryInterval.String())
	if since < maxTime {
		return operation, retryInterval, nil
	}
	log.Errorf("Aborting after %s of failing retries", maxTime.String())
	return om.OperationFailed(operation, errorMessage, log)
}

func (om *ProvisionOperationManager) HandleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return om.OperationFailed(operation, msg, log)
}

func (om *ProvisionOperationManager) update(operation internal.ProvisioningOperation, state domain.LastOperationState, description string, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration) {
	return om.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
		operation.State = state
		operation.Description = fmt.Sprintf("%s : %s", operation.Description, description)
	}, log)
}
