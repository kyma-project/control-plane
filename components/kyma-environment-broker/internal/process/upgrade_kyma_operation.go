package process

import (
	"errors"
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v7/domain"
)

type UpgradeKymaOperationManager struct {
	storage storage.UpgradeKyma
}

func NewUpgradeKymaOperationManager(storage storage.Operations) *UpgradeKymaOperationManager {
	return &UpgradeKymaOperationManager{storage: storage}
}

// OperationSucceeded marks the operation as succeeded and only repeats it if there is a storage error
func (om *UpgradeKymaOperationManager) OperationSucceeded(operation internal.UpgradeKymaOperation, description string) (internal.UpgradeKymaOperation, time.Duration, error) {
	updatedOperation, repeat := om.update(operation, domain.Succeeded, description)
	// repeat in case of storage error
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	return updatedOperation, 0, nil
}

// OperationFailed marks the operation as failed and only repeats it if there is a storage error
func (om *UpgradeKymaOperationManager) OperationFailed(operation internal.UpgradeKymaOperation, description string) (internal.UpgradeKymaOperation, time.Duration, error) {
	updatedOperation, repeat := om.update(operation, domain.Failed, description)
	// repeat in case of storage error
	if repeat != 0 {
		return updatedOperation, repeat, nil
	}

	return updatedOperation, 0, errors.New(description)
}

// UpdateOperation updates a given operation
func (om *UpgradeKymaOperationManager) UpdateOperation(operation internal.UpgradeKymaOperation) (internal.UpgradeKymaOperation, time.Duration) {
	updatedOperation, err := om.storage.UpdateUpgradeKymaOperation(operation)
	if err != nil {
		return operation, 1 * time.Minute
	}
	return *updatedOperation, 0
}

func (om *UpgradeKymaOperationManager) update(operation internal.UpgradeKymaOperation, state domain.LastOperationState, description string) (internal.UpgradeKymaOperation, time.Duration) {
	operation.State = state
	operation.Description = fmt.Sprintf("%s : %s", operation.Description, description)

	return om.UpdateOperation(operation)
}
