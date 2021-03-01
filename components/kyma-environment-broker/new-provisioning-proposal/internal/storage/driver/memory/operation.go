package memory

import (
	"sync"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/new-provisioning-proposal/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/new-provisioning-proposal/internal/storage/dberr"
)

// OperationType defines the possible types of an asynchronous operation to a broker.
type OperationType string

const (
	// OperationTypeProvision means provisioning OperationType
	OperationTypeProvision OperationType = "provision"
	// OperationTypeDeprovision means deprovision OperationType
	OperationTypeDeprovision OperationType = "deprovision"
	// OperationTypeUndefined means undefined OperationType
	OperationTypeUndefined OperationType = ""
	// OperationTypeUpgradeKyma means upgrade Kyma OperationType
	OperationTypeUpgradeKyma OperationType = "upgradeKyma"
)

type operations struct {
	mu sync.Mutex

	provisioningOperations map[string]internal.ProvisioningOperation
}

// NewOperation creates in-memory storage for OSB operations.
func NewOperation() *operations {
	return &operations{
		provisioningOperations: make(map[string]internal.ProvisioningOperation, 0),
	}
}

func (s *operations) InsertProvisioningOperation(operation internal.ProvisioningOperation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := operation.ID
	if _, exists := s.provisioningOperations[id]; exists {
		return dberr.AlreadyExists("instance operation with id %s already exist", id)
	}

	s.provisioningOperations[id] = operation
	return nil
}

func (s *operations) GetProvisioningOperationByID(operationID string) (*internal.ProvisioningOperation, error) {
	op, exists := s.provisioningOperations[operationID]
	if !exists {
		return nil, dberr.NotFound("instance provisioning operation with id %s not found", operationID)
	}
	return &op, nil
}

func (s *operations) GetProvisioningOperationByInstanceID(instanceID string) (*internal.ProvisioningOperation, error) {
	for _, op := range s.provisioningOperations {
		if op.InstanceID == instanceID {
			return &op, nil
		}
	}
	return nil, dberr.NotFound("instance provisioning operation with instanceID %s not found", instanceID)
}

func (s *operations) UpdateProvisioningOperation(op internal.ProvisioningOperation) (*internal.ProvisioningOperation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldOp, exists := s.provisioningOperations[op.ID]
	if !exists {
		return nil, dberr.NotFound("instance operation with id %s not found", op.ID)
	}
	if oldOp.Version != op.Version {
		return nil, dberr.Conflict("unable to update provisioning operation with id %s (for instance id %s) - conflict", op.ID, op.InstanceID)
	}
	op.Version = op.Version + 1
	s.provisioningOperations[op.ID] = op

	return &op, nil
}

func (s *operations) GetOperationByID(operationID string) (*internal.Operation, error) {
	var res *internal.Operation

	provisionOp, exists := s.provisioningOperations[operationID]
	if exists {
		res = &provisionOp.Operation
	}
	if res == nil {
		return nil, dberr.NotFound("instance operation with id %s not found", operationID)
	}

	return res, nil
}
