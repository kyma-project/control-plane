package process

import (
	"errors"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/sirupsen/logrus"
)

type UpgradeKymaOperationManager struct {
	storage storage.UpgradeKyma
}

func NewUpgradeKymaOperationManager(storage storage.Operations) *UpgradeKymaOperationManager {
	return &UpgradeKymaOperationManager{storage: storage}
}

// OperationSucceeded marks the operation as succeeded and only repeats it if there is a storage error
func (om *UpgradeKymaOperationManager) OperationSucceeded(operation internal.UpgradeKymaOperation, description string) (internal.UpgradeKymaOperation, time.Duration, error) {
	updatedOperation, repeat := om.update(operation, orchestration.Succeeded, description)
	// repeat in case of storage error
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	return updatedOperation, 0, nil
}

// OperationFailed marks the operation as failed and only repeats it if there is a storage error
func (om *UpgradeKymaOperationManager) OperationFailed(operation internal.UpgradeKymaOperation, description string) (internal.UpgradeKymaOperation, time.Duration, error) {
	updatedOperation, repeat := om.update(operation, orchestration.Failed, description)
	// repeat in case of storage error
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	return updatedOperation, 0, errors.New(description)
}

// OperationSucceeded marks the operation as succeeded and only repeats it if there is a storage error
func (om *UpgradeKymaOperationManager) OperationCanceled(operation internal.UpgradeKymaOperation, description string) (internal.UpgradeKymaOperation, time.Duration, error) {
	updatedOperation, repeat := om.update(operation, orchestration.Canceled, description)
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	return updatedOperation, 0, nil
}

// RetryOperation retries an operation for at maxTime in retryInterval steps and fails the operation if retrying failed
func (om *UpgradeKymaOperationManager) RetryOperation(operation internal.UpgradeKymaOperation, errorMessage string, retryInterval time.Duration, maxTime time.Duration, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	since := time.Since(operation.UpdatedAt)

	log.Infof("Retry Operation was triggered with message: %s", errorMessage)
	log.Infof("Retrying for %s in %s steps", maxTime.String(), retryInterval.String())
	if since < maxTime {
		return operation, retryInterval, nil
	}
	log.Errorf("Aborting after %s of failing retries", maxTime.String())
	return om.OperationFailed(operation, errorMessage)
}

// UpdateOperation updates a given operation
func (om *UpgradeKymaOperationManager) UpdateOperation(operation internal.UpgradeKymaOperation) (internal.UpgradeKymaOperation, time.Duration) {
	updatedOperation, err := om.storage.UpdateUpgradeKymaOperation(operation)
	if err != nil {
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
	updatedOperation, repeat := om.update(operation, domain.InProgress, description)
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	log.Errorf("Omitting after %s of failing retries", maxTime.String())
	return updatedOperation, 0, nil
}


func (om *UpgradeKymaOperationManager) update(operation internal.UpgradeKymaOperation, state domain.LastOperationState, description string) (internal.UpgradeKymaOperation, time.Duration) {
	operation.State = state
	operation.Description = description

	return om.UpdateOperation(operation)
}
