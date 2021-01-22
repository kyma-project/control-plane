package memory

import (
	"sort"
	"sync"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/pagination"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"

	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/pkg/errors"
)

type operations struct {
	mu sync.Mutex

	provisioningOperations   map[string]internal.ProvisioningOperation
	deprovisioningOperations map[string]internal.DeprovisioningOperation
	upgradeKymaOperations    map[string]internal.UpgradeKymaOperation
}

// NewOperation creates in-memory storage for OSB operations.
func NewOperation() *operations {
	return &operations{
		provisioningOperations:   make(map[string]internal.ProvisioningOperation, 0),
		deprovisioningOperations: make(map[string]internal.DeprovisioningOperation, 0),
		upgradeKymaOperations:    make(map[string]internal.UpgradeKymaOperation, 0),
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

func (s *operations) InsertDeprovisioningOperation(operation internal.DeprovisioningOperation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := operation.ID
	if _, exists := s.deprovisioningOperations[id]; exists {
		return dberr.AlreadyExists("instance operation with id %s already exist", id)
	}

	s.deprovisioningOperations[id] = operation
	return nil
}

func (s *operations) GetDeprovisioningOperationByID(operationID string) (*internal.DeprovisioningOperation, error) {
	op, exists := s.deprovisioningOperations[operationID]
	if !exists {
		return nil, dberr.NotFound("instance deprovisioning operation with id %s not found", operationID)
	}
	return &op, nil
}

func (s *operations) GetDeprovisioningOperationByInstanceID(instanceID string) (*internal.DeprovisioningOperation, error) {
	for _, op := range s.deprovisioningOperations {
		if op.InstanceID == instanceID {
			return &op, nil
		}
	}

	return nil, dberr.NotFound("instance deprovisioning operation with instanceID %s not found", instanceID)
}

func (s *operations) UpdateDeprovisioningOperation(op internal.DeprovisioningOperation) (*internal.DeprovisioningOperation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldOp, exists := s.deprovisioningOperations[op.ID]
	if !exists {
		return nil, dberr.NotFound("instance operation with id %s not found", op.ID)
	}
	if oldOp.Version != op.Version {
		return nil, dberr.Conflict("unable to update deprovisioning operation with id %s (for instance id %s) - conflict", op.ID, op.InstanceID)
	}
	op.Version = op.Version + 1
	s.deprovisioningOperations[op.ID] = op

	return &op, nil
}

func (s *operations) InsertUpgradeKymaOperation(operation internal.UpgradeKymaOperation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := operation.Operation.ID
	if _, exists := s.upgradeKymaOperations[id]; exists {
		return dberr.AlreadyExists("instance operation with id %s already exist", id)
	}

	s.upgradeKymaOperations[id] = operation
	return nil
}

func (s *operations) GetUpgradeKymaOperationByID(operationID string) (*internal.UpgradeKymaOperation, error) {
	op, exists := s.upgradeKymaOperations[operationID]
	if !exists {
		return nil, dberr.NotFound("instance upgradeKyma operation with id %s not found", operationID)
	}
	return &op, nil
}

func (s *operations) GetUpgradeKymaOperationByInstanceID(instanceID string) (*internal.UpgradeKymaOperation, error) {
	for _, op := range s.upgradeKymaOperations {
		if op.InstanceID == instanceID {
			return &op, nil
		}
	}

	return nil, dberr.NotFound("instance upgradeKyma operation with instanceID %s not found", instanceID)
}

func (s *operations) UpdateUpgradeKymaOperation(op internal.UpgradeKymaOperation) (*internal.UpgradeKymaOperation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldOp, exists := s.upgradeKymaOperations[op.Operation.ID]
	if !exists {
		return nil, dberr.NotFound("instance operation with id %s not found", op.Operation.ID)
	}
	if oldOp.Version != op.Version {
		return nil, dberr.Conflict("unable to update upgradeKyma operation with id %s (for instance id %s) - conflict", op.Operation.ID, op.InstanceID)
	}
	op.Version = op.Version + 1
	s.upgradeKymaOperations[op.Operation.ID] = op

	return &op, nil
}

func (s *operations) GetLastOperation(instanceID string) (*internal.Operation, error) {
	var rows []internal.Operation

	for _, op := range s.provisioningOperations {
		if op.InstanceID == instanceID && op.State != orchestration.Pending {
			rows = append(rows, op.Operation)
		}
	}
	for _, op := range s.deprovisioningOperations {
		if op.InstanceID == instanceID && op.State != orchestration.Pending {
			rows = append(rows, op.Operation)
		}
	}
	for _, op := range s.upgradeKymaOperations {
		if op.InstanceID == instanceID && op.State != orchestration.Pending {
			rows = append(rows, op.Operation)
		}
	}

	if len(rows) == 0 {
		return nil, dberr.NotFound("instance operation with instance_id %s not found", instanceID)
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
	})

	return &rows[0], nil
}

func (s *operations) GetOperationByID(operationID string) (*internal.Operation, error) {
	var res *internal.Operation

	provisionOp, exists := s.provisioningOperations[operationID]
	if exists {
		res = &provisionOp.Operation
	}
	deprovisionOp, exists := s.deprovisioningOperations[operationID]
	if exists {
		res = &deprovisionOp.Operation
	}
	upgradeKymaOp, exists := s.upgradeKymaOperations[operationID]
	if exists {
		res = &upgradeKymaOp.Operation
	}
	if res == nil {
		return nil, dberr.NotFound("instance operation with id %s not found", operationID)
	}

	return res, nil
}

func (s *operations) GetOperationsInProgressByType(opType dbmodel.OperationType) ([]internal.Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ops := make([]internal.Operation, 0)
	switch opType {
	case dbmodel.OperationTypeProvision:
		for _, op := range s.provisioningOperations {
			if op.State == domain.InProgress {
				ops = append(ops, op.Operation)
			}
		}
	case dbmodel.OperationTypeDeprovision:
		for _, op := range s.deprovisioningOperations {
			if op.State == domain.InProgress {
				ops = append(ops, op.Operation)
			}
		}
	}

	return ops, nil
}

func (s *operations) GetOperationsForIDs(opIdList []string) ([]internal.Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ops := make([]internal.Operation, 0)
	for _, opID := range opIdList {
		for _, op := range s.upgradeKymaOperations {
			if op.Operation.ID == opID {
				ops = append(ops, op.Operation)
			}
		}
	}

	for _, opID := range opIdList {
		for _, op := range s.provisioningOperations {
			if op.Operation.ID == opID {
				ops = append(ops, op.Operation)
			}
		}
	}

	for _, opID := range opIdList {
		for _, op := range s.deprovisioningOperations {
			if op.Operation.ID == opID {
				ops = append(ops, op.Operation)
			}
		}
	}

	if len(ops) == 0 {
		return nil, dberr.NotFound("operations with ids from list %+q not exist", opIdList)
	}

	return ops, nil
}

func (s *operations) GetOperationStatsByPlan() (map[string]internal.OperationStats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string]internal.OperationStats)

	for _, op := range s.provisioningOperations {
		if op.ProvisioningParameters.PlanID == "" {
			continue
		}
		if _, ok := result[op.ProvisioningParameters.PlanID]; !ok {
			result[op.ProvisioningParameters.PlanID] = internal.OperationStats{
				Provisioning:   make(map[domain.LastOperationState]int),
				Deprovisioning: make(map[domain.LastOperationState]int),
			}
		}
		result[op.ProvisioningParameters.PlanID].Provisioning[op.State] += 1
	}
	for _, op := range s.deprovisioningOperations {
		if op.ProvisioningParameters.PlanID == "" {
			continue
		}
		if _, ok := result[op.ProvisioningParameters.PlanID]; !ok {
			result[op.ProvisioningParameters.PlanID] = internal.OperationStats{
				Provisioning:   make(map[domain.LastOperationState]int),
				Deprovisioning: make(map[domain.LastOperationState]int),
			}
		}
		result[op.ProvisioningParameters.PlanID].Deprovisioning[op.State] += 1
	}
	return result, nil
}

func (s *operations) GetOperationStatsForOrchestration(orchestrationID string) (map[string]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := map[string]int{
		orchestration.Canceled:   0,
		orchestration.Canceling:  0,
		orchestration.InProgress: 0,
		orchestration.Pending:    0,
		orchestration.Succeeded:  0,
		orchestration.Failed:     0,
	}
	for _, op := range s.upgradeKymaOperations {
		if op.OrchestrationID == orchestrationID {
			result[string(op.State)] = result[string(op.State)] + 1
		}
	}
	return result, nil
}

func (s *operations) ListOperations(filter dbmodel.OperationFilter) ([]internal.Operation, int, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]internal.Operation, 0)
	offset := pagination.ConvertPageAndPageSizeToOffset(filter.PageSize, filter.Page)

	operations, err := s.filterAll(filter)
	if err != nil {
		return nil, 0, 0, errors.Wrap(err, "while listing operations")
	}
	s.sortByCreatedAt(operations)

	for i := offset; (filter.PageSize < 1 || i < offset+filter.PageSize) && i < len(operations)+offset; i++ {
		result = append(result, operations[i])
	}

	return result,
		len(result),
		len(operations),
		nil
}

func (s *operations) ListUpgradeKymaOperations() ([]internal.UpgradeKymaOperation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Empty filter means get all
	operations := s.filterUpgrade(dbmodel.OperationFilter{})
	s.sortUpgradeByCreatedAt(operations)

	return operations, nil
}

func (s *operations) ListUpgradeKymaOperationsByOrchestrationID(orchestrationID string, filter dbmodel.OperationFilter) ([]internal.UpgradeKymaOperation, int, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]internal.UpgradeKymaOperation, 0)
	offset := pagination.ConvertPageAndPageSizeToOffset(filter.PageSize, filter.Page)

	operations := s.filterUpgrade(filter)
	s.sortUpgradeByCreatedAt(operations)

	for i := offset; (filter.PageSize < 1 || i < offset+filter.PageSize) && i < len(operations)+offset; i++ {
		result = append(result, s.upgradeKymaOperations[operations[i].OrchestrationID])
	}

	return result,
		len(result),
		len(operations),
		nil
}

func (s *operations) ListUpgradeKymaOperationsByInstanceID(instanceID string) ([]internal.UpgradeKymaOperation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Empty filter means get all
	operations := s.filterUpgrade(dbmodel.OperationFilter{})
	s.sortUpgradeByCreatedAt(operations)

	return operations, nil
}

func (s *operations) sortUpgradeByCreatedAt(operations []internal.UpgradeKymaOperation) {
	sort.Slice(operations, func(i, j int) bool {
		return operations[i].CreatedAt.Before(operations[j].CreatedAt)
	})
}

func (s *operations) sortByCreatedAt(operations []internal.Operation) {
	sort.Slice(operations, func(i, j int) bool {
		return operations[i].CreatedAt.Before(operations[j].CreatedAt)
	})
}

func (s *operations) getOperation(id string) (internal.Operation, error) {
	for _, op := range s.upgradeKymaOperations {
		if op.Operation.ID == id {
			return op.Operation, nil
		}
	}
	for _, op := range s.provisioningOperations {
		if op.Operation.ID == id {
			return op.Operation, nil
		}
	}
	for _, op := range s.deprovisioningOperations {
		if op.Operation.ID == id {
			return op.Operation, nil
		}
	}

	return internal.Operation{}, dberr.NotFound("operation not found")
}

func (s *operations) updateOperation(operation internal.Operation) (internal.Operation, error) {
	for i, op := range s.upgradeKymaOperations {
		if op.Operation.ID == operation.ID {
			temp := s.upgradeKymaOperations[i]
			temp.ProvisioningParameters = operation.ProvisioningParameters
			s.upgradeKymaOperations[i] = temp
			return operation, nil
		}
	}
	for i, op := range s.provisioningOperations {
		if op.Operation.ID == operation.ID {
			temp := s.provisioningOperations[i]
			temp.ProvisioningParameters = operation.ProvisioningParameters
			s.provisioningOperations[i] = temp
			return operation, nil
		}
	}
	for i, op := range s.deprovisioningOperations {
		if op.Operation.ID == operation.ID {
			temp := s.deprovisioningOperations[i]
			temp.ProvisioningParameters = operation.ProvisioningParameters
			s.deprovisioningOperations[i] = temp
			return operation, nil
		}
	}
	return internal.Operation{}, dberr.NotFound("operation not found")
}

func (s *operations) getAll() ([]internal.Operation, error) {
	ops := make([]internal.Operation, 0)
	for _, op := range s.upgradeKymaOperations {
		ops = append(ops, op.Operation)
	}
	for _, op := range s.provisioningOperations {
		ops = append(ops, op.Operation)
	}
	for _, op := range s.deprovisioningOperations {
		ops = append(ops, op.Operation)
	}
	if len(ops) == 0 {
		return nil, dberr.NotFound("operations not found")
	}

	return ops, nil
}

func (s *operations) filterAll(filter dbmodel.OperationFilter) ([]internal.Operation, error) {
	result := make([]internal.Operation, 0)
	ops, err := s.getAll()
	if err != nil {
		return nil, err
	}
	for _, op := range ops {
		if ok := matchFilter(string(op.State), filter.States, s.equalFilter); !ok {
			continue
		}
		result = append(result, op)
	}
	return result, nil
}

func (s *operations) filterUpgrade(filter dbmodel.OperationFilter) []internal.UpgradeKymaOperation {
	operations := make([]internal.UpgradeKymaOperation, 0, len(s.upgradeKymaOperations))
	for _, v := range s.upgradeKymaOperations {
		if ok := matchFilter(string(v.State), filter.States, s.equalFilter); !ok {
			continue
		}

		operations = append(operations, v)
	}

	return operations
}

func (s *operations) equalFilter(a, b string) bool {
	return a == b
}
