package process

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type UpgradeClusterOperationManager struct {
	storage storage.UpgradeCluster
}

func NewUpgradeClusterOperationManager(storage storage.Operations) *UpgradeClusterOperationManager {
	return &UpgradeClusterOperationManager{storage: storage}
}

// OperationSucceeded marks the operation as succeeded and only repeats it if there is a storage error
func (om *UpgradeClusterOperationManager) OperationSucceeded(operation internal.UpgradeClusterOperation, description string) (internal.UpgradeClusterOperation, time.Duration, error) {
	updatedOperation, repeat := om.update(operation, orchestration.Succeeded, description)
	// repeat in case of storage error
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	return updatedOperation, 0, nil
}

// OperationFailed marks the operation as failed and only repeats it if there is a storage error
func (om *UpgradeClusterOperationManager) OperationFailed(operation internal.UpgradeClusterOperation, description string) (internal.UpgradeClusterOperation, time.Duration, error) {
	updatedOperation, repeat := om.update(operation, orchestration.Failed, description)
	// repeat in case of storage error
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	return updatedOperation, 0, errors.New(description)
}

// OperationSucceeded marks the operation as succeeded and only repeats it if there is a storage error
func (om *UpgradeClusterOperationManager) OperationCanceled(operation internal.UpgradeClusterOperation, description string) (internal.UpgradeClusterOperation, time.Duration, error) {
	updatedOperation, repeat := om.update(operation, orchestration.Canceled, description)
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	return updatedOperation, 0, nil
}

// RetryOperation retries an operation for at maxTime in retryInterval steps and fails the operation if retrying failed
func (om *UpgradeClusterOperationManager) RetryOperation(operation internal.UpgradeClusterOperation, errorMessage string, retryInterval time.Duration, maxTime time.Duration, log logrus.FieldLogger) (internal.UpgradeClusterOperation, time.Duration, error) {
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
func (om *UpgradeClusterOperationManager) UpdateOperation(operation internal.UpgradeClusterOperation) (internal.UpgradeClusterOperation, time.Duration) {
	updatedOperation, err := om.storage.UpdateUpgradeClusterOperation(operation)
	if err != nil {
		logrus.WithField("orchestrationID", operation.OrchestrationID).
			WithField("instanceID", operation.InstanceID).
			Errorf("Update provisioning operation failed: %s", err.Error())
		return operation, 1 * time.Minute
	}
	return *updatedOperation, 0
}

// RetryOperationWithoutFail retries an operation for at maxTime in retryInterval steps and omits the operation if retrying failed
func (om *UpgradeClusterOperationManager) RetryOperationWithoutFail(operation internal.UpgradeClusterOperation, description string, retryInterval, maxTime time.Duration, log logrus.FieldLogger) (internal.UpgradeClusterOperation, time.Duration, error) {
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

func (om *UpgradeClusterOperationManager) update(operation internal.UpgradeClusterOperation, state domain.LastOperationState, description string) (internal.UpgradeClusterOperation, time.Duration) {
	operation.State = state
	operation.Description = description

	return om.UpdateOperation(operation)
}
