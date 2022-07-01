package postsql

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"

	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	Retrying  = "retrying" // to signal a retry sign before marking it to pending
	Succeeded = "succeeded"
	Failed    = "failed"
)

type operations struct {
	postsql.Factory
	cipher Cipher
}

func NewOperation(sess postsql.Factory, cipher Cipher) *operations {
	return &operations{
		Factory: sess,
		cipher:  cipher,
	}
}

// InsertProvisioningOperation insert new ProvisioningOperation to storage
func (s *operations) InsertProvisioningOperation(operation internal.ProvisioningOperation) error {
	dto, err := s.provisioningOperationToDTO(&operation)
	if err != nil {
		return errors.Wrapf(err, "while inserting provisioning operation (id: %s)", operation.ID)
	}

	return s.insert(dto)
}

// InsertOperation insert new Operation to storage
func (s *operations) InsertOperation(operation internal.Operation) error {
	dto, err := s.operationToDTO(&operation)

	if err != nil {
		return errors.Wrapf(err, "while inserting operation (id: %s)", operation.ID)
	}

	return s.insert(dto)
}

// GetOperationByInstanceID fetches the latest Operation by given instanceID, returns error if not found
func (s *operations) GetOperationByInstanceID(instanceID string) (*internal.Operation, error) {

	op, err := s.getByInstanceID(instanceID)
	if err != nil {
		return nil, err
	}

	var operation internal.Operation
	err = json.Unmarshal([]byte(op.Data), &operation)
	if err != nil {
		return nil, errors.New("unable to unmarshall provisioning data")
	}

	ret, err := s.toOperation(op, operation.InstanceDetails)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return &ret, nil
}

// GetProvisioningOperationByID fetches the ProvisioningOperation by given ID, returns error if not found
func (s *operations) GetProvisioningOperationByID(operationID string) (*internal.ProvisioningOperation, error) {
	operation, err := s.getByID(operationID)
	if err != nil {
		return nil, errors.Wrap(err, "while getting operation by ID")
	}

	ret, err := s.toProvisioningOperation(operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// GetProvisioningOperationByInstanceID fetches the latest ProvisioningOperation by given instanceID, returns error if not found
func (s *operations) GetProvisioningOperationByInstanceID(instanceID string) (*internal.ProvisioningOperation, error) {

	operation, err := s.getByTypeAndInstanceID(instanceID, internal.OperationTypeProvision)
	if err != nil {
		return nil, err
	}
	ret, err := s.toProvisioningOperation(operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// UpdateOperation updates Operation, fails if not exists or optimistic locking failure occurs.
func (s *operations) UpdateOperation(op internal.Operation) (*internal.Operation, error) {
	op.UpdatedAt = time.Now()
	dto, err := s.operationToDTO(&op)

	if err != nil {
		return nil, errors.Wrapf(err, "while converting Operation to DTO")
	}

	lastErr := s.update(dto)
	op.Version = op.Version + 1

	return &op, lastErr
}

// UpdateProvisioningOperation updates ProvisioningOperation, fails if not exists or optimistic locking failure occurs.
func (s *operations) UpdateProvisioningOperation(op internal.ProvisioningOperation) (*internal.ProvisioningOperation, error) {
	op.UpdatedAt = time.Now()
	dto, err := s.provisioningOperationToDTO(&op)

	if err != nil {
		return nil, errors.Wrapf(err, "while converting Operation to DTO")
	}

	lastErr := s.update(dto)
	op.Version = op.Version + 1

	return &op, lastErr
}

func (s *operations) ListProvisioningOperationsByInstanceID(instanceID string) ([]internal.ProvisioningOperation, error) {

	operations, err := s.listOperationsByInstanceIdAndType(instanceID, internal.OperationTypeProvision)
	if err != nil {
		return nil, errors.Wrapf(err, "while loading operations list")
	}

	ret, err := s.toProvisioningOperationList(operations)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

func (s *operations) ListOperationsByInstanceID(instanceID string) ([]internal.Operation, error) {

	operations, err := s.listOperationsByInstanceId(instanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "while loading operations list")
	}

	ret, err := s.toOperationList(operations)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// InsertDeprovisioningOperation insert new DeprovisioningOperation to storage
func (s *operations) InsertDeprovisioningOperation(operation internal.DeprovisioningOperation) error {

	dto, err := s.deprovisioningOperationToDTO(&operation)
	if err != nil {
		return errors.Wrapf(err, "while converting Operation to DTO")
	}

	return s.insert(dto)
}

// GetDeprovisioningOperationByID fetches the DeprovisioningOperation by given ID, returns error if not found
func (s *operations) GetDeprovisioningOperationByID(operationID string) (*internal.DeprovisioningOperation, error) {
	operation, err := s.getByID(operationID)
	if err != nil {
		return nil, errors.Wrap(err, "while getting operation by ID")
	}

	ret, err := s.toDeprovisioningOperation(operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// GetDeprovisioningOperationByInstanceID fetches the latest DeprovisioningOperation by given instanceID, returns error if not found
func (s *operations) GetDeprovisioningOperationByInstanceID(instanceID string) (*internal.DeprovisioningOperation, error) {
	operation, err := s.getByTypeAndInstanceID(instanceID, internal.OperationTypeDeprovision)
	if err != nil {
		return nil, err
	}
	ret, err := s.toDeprovisioningOperation(operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// UpdateDeprovisioningOperation updates DeprovisioningOperation, fails if not exists or optimistic locking failure occurs.
func (s *operations) UpdateDeprovisioningOperation(operation internal.DeprovisioningOperation) (*internal.DeprovisioningOperation, error) {
	operation.UpdatedAt = time.Now()

	dto, err := s.deprovisioningOperationToDTO(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting Operation to DTO")
	}

	lastErr := s.update(dto)
	operation.Version = operation.Version + 1
	return &operation, lastErr
}

// ListDeprovisioningoOperationsByInstanceID
func (s *operations) ListDeprovisioningOperationsByInstanceID(instanceID string) ([]internal.DeprovisioningOperation, error) {
	operations, err := s.listOperationsByInstanceIdAndType(instanceID, internal.OperationTypeDeprovision)
	if err != nil {
		return nil, err
	}

	ret, err := s.toDeprovisioningOperationList(operations)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// ListDeprovisioningOperations lists deprovisioning operations
func (s *operations) ListDeprovisioningOperations() ([]internal.DeprovisioningOperation, error) {
	var lastErr dberr.Error

	operations, err := s.listOperationsByType(internal.OperationTypeDeprovision)
	if err != nil {
		return nil, lastErr
	}

	ret, err := s.toDeprovisioningOperationList(operations)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// InsertUpgradeKymaOperation insert new UpgradeKymaOperation to storage
func (s *operations) InsertUpgradeKymaOperation(operation internal.UpgradeKymaOperation) error {
	dto, err := s.upgradeKymaOperationToDTO(&operation)
	if err != nil {
		return errors.Wrapf(err, "while inserting upgrade kyma operation (id: %s)", operation.Operation.ID)
	}

	return s.insert(dto)
}

// GetUpgradeKymaOperationByID fetches the UpgradeKymaOperation by given ID, returns error if not found
func (s *operations) GetUpgradeKymaOperationByID(operationID string) (*internal.UpgradeKymaOperation, error) {
	operation, err := s.getByID(operationID)
	if err != nil {
		return nil, errors.Wrap(err, "while getting operation by ID")
	}

	ret, err := s.toUpgradeKymaOperation(operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// GetUpgradeKymaOperationByInstanceID fetches the latest UpgradeKymaOperation by given instanceID, returns error if not found
func (s *operations) GetUpgradeKymaOperationByInstanceID(instanceID string) (*internal.UpgradeKymaOperation, error) {
	operation, err := s.getByTypeAndInstanceID(instanceID, internal.OperationTypeUpgradeKyma)
	if err != nil {
		return nil, err
	}
	ret, err := s.toUpgradeKymaOperation(operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

func (s *operations) ListUpgradeKymaOperations() ([]internal.UpgradeKymaOperation, error) {
	var operations []dbmodel.OperationDTO

	operations, err := s.listOperationsByType(internal.OperationTypeUpgradeKyma)
	if err != nil {
		return nil, err
	}
	ret, err := s.toUpgradeKymaOperationList(operations)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

func (s *operations) ListUpgradeKymaOperationsByInstanceID(instanceID string) ([]internal.UpgradeKymaOperation, error) {
	operations, err := s.listOperationsByInstanceIdAndType(instanceID, internal.OperationTypeUpgradeKyma)
	if err != nil {
		return nil, err
	}

	ret, err := s.toUpgradeKymaOperationList(operations)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// UpdateUpgradeKymaOperation updates UpgradeKymaOperation, fails if not exists or optimistic locking failure occurs.
func (s *operations) UpdateUpgradeKymaOperation(operation internal.UpgradeKymaOperation) (*internal.UpgradeKymaOperation, error) {
	operation.UpdatedAt = time.Now()
	dto, err := s.upgradeKymaOperationToDTO(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting Operation to DTO")
	}

	err = s.update(dto)
	operation.Version = operation.Version + 1
	return &operation, err
}

// GetLastOperation returns Operation for given instance ID which is not in 'pending' state. Returns an error if the operation does not exists.
func (s *operations) GetLastOperation(instanceID string) (*internal.Operation, error) {
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}
	op := internal.Operation{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetLastOperation(instanceID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("Operation with instance_id %s not exist", instanceID)
				return false, lastErr
			}
			log.Errorf("while reading operation from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	err = json.Unmarshal([]byte(operation.Data), &op)
	if err != nil {
		return nil, errors.New("unable to unmarshall operation data")
	}
	op, err = s.toOperation(&operation, op.InstanceDetails)
	if err != nil {
		return nil, err
	}
	return &op, nil
}

// GetOperationByID returns Operation with given ID. Returns an error if the operation does not exists.
func (s *operations) GetOperationByID(operationID string) (*internal.Operation, error) {
	op := internal.Operation{}
	dto, err := s.getByID(operationID)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(dto.Data), &op)
	if err != nil {
		return nil, errors.New("unable to unmarshall operation data")
	}

	op, err = s.toOperation(dto, op.InstanceDetails)
	if err != nil {
		return nil, err
	}
	return &op, nil
}

func (s *operations) GetNotFinishedOperationsByType(operationType internal.OperationType) ([]internal.Operation, error) {
	session := s.NewReadSession()
	operations := make([]dbmodel.OperationDTO, 0)
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		dto, err := session.GetNotFinishedOperationsByType(operationType)
		if err != nil {
			log.Errorf("while getting operations from the storage: %v", err)
			return false, nil
		}
		operations = dto
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return s.toOperations(operations)
}

func (s *operations) GetOperationStatsByPlan() (map[string]internal.OperationStats, error) {
	entries, err := s.NewReadSession().GetOperationStats()
	if err != nil {
		return nil, err
	}
	result := make(map[string]internal.OperationStats)

	for _, e := range entries {
		if e.PlanID == "" {
			continue
		}
		if _, ok := result[e.PlanID]; !ok {
			result[e.PlanID] = internal.OperationStats{
				Provisioning:   make(map[domain.LastOperationState]int),
				Deprovisioning: make(map[domain.LastOperationState]int),
			}
		}
		switch internal.OperationType(e.Type) {
		case internal.OperationTypeProvision:
			result[e.PlanID].Provisioning[domain.LastOperationState(e.State)] += 1
		case internal.OperationTypeDeprovision:
			result[e.PlanID].Deprovisioning[domain.LastOperationState(e.State)] += 1
		}
	}
	return result, nil
}

func (s *operations) GetOperationStatsForOrchestration(orchestrationID string) (map[string]int, error) {
	entries, err := s.NewReadSession().GetOperationStatsForOrchestration(orchestrationID)
	if err != nil {
		return map[string]int{}, err
	}
	fmt.Println("GetOperationStatsForOrchestration entries:", entries)

	result := make(map[string]int)
	resultPerInstanceID := make(map[string][]string)

	for _, entry := range entries {
		resultPerInstanceID[entry.InstanceID] = append(resultPerInstanceID[entry.InstanceID], entry.State)
		fmt.Println("set resultPerInstanceID", entry.InstanceID, resultPerInstanceID[entry.InstanceID])
	}

	fmt.Println("show resultPerInstanceID", resultPerInstanceID)
	var invalidFailed, failedFound bool

	for instanceID, statuses := range resultPerInstanceID {
		fmt.Println("loop resultPerInstanceID", instanceID, statuses)

		invalidFailed = false
		failedFound = false
		for _, status := range statuses {
			if status == Failed {
				failedFound = true
			}
			if status == Succeeded || status == Retrying {
				fmt.Println("found invalidFailed status:=", status)
				invalidFailed = true
			}
		}
		if failedFound && !invalidFailed {
			result[Failed] += 1
		}
	}

	for _, entry := range entries {
		if entry.State != Failed {
			result[entry.State] += 1
			fmt.Println("loop entries", entry.State, result[entry.State])
		}
	}
	return result, nil
}

func (s *operations) GetOperationsForIDs(operationIDList []string) ([]internal.Operation, error) {
	session := s.NewReadSession()
	operations := make([]dbmodel.OperationDTO, 0)
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		dto, err := session.GetOperationsForIDs(operationIDList)
		if err != nil {
			log.Errorf("while getting operations from the storage: %v", err)
			return false, nil
		}
		operations = dto
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return s.toOperations(operations)
}

func (s *operations) ListOperations(filter dbmodel.OperationFilter) ([]internal.Operation, int, int, error) {
	session := s.NewReadSession()

	var (
		lastErr     error
		size, total int
		operations  = make([]dbmodel.OperationDTO, 0)
	)

	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operations, size, total, lastErr = session.ListOperations(filter)
		if lastErr != nil {
			log.Errorf("while getting operations from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, -1, -1, err
	}

	result, err := s.toOperations(operations)

	return result, size, total, err
}

func (s *operations) ListUpgradeKymaOperationsByOrchestrationID(orchestrationID string, filter dbmodel.OperationFilter) ([]internal.UpgradeKymaOperation, int, int, error) {
	session := s.NewReadSession()
	var (
		operations        = make([]dbmodel.OperationDTO, 0)
		lastErr           error
		count, totalCount int
	)
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operations, count, totalCount, lastErr = session.ListOperationsByOrchestrationID(orchestrationID, filter)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("Operations for orchestration ID %s not exist", orchestrationID)
				return false, lastErr
			}
			log.Errorf("while reading operation from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, -1, -1, errors.Wrapf(err, "while getting operation by ID: %v", lastErr)
	}
	ret, err := s.toUpgradeKymaOperationList(operations)
	if err != nil {
		return nil, -1, -1, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, count, totalCount, nil
}

func (s *operations) ListOperationsByOrchestrationID(orchestrationID string, filter dbmodel.OperationFilter) ([]internal.Operation, int, int, error) {
	session := s.NewReadSession()
	var (
		operations        = make([]dbmodel.OperationDTO, 0)
		lastErr           error
		count, totalCount int
	)
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operations, count, totalCount, lastErr = session.ListOperationsByOrchestrationID(orchestrationID, filter)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("Operations for orchestration ID %s not exist", orchestrationID)
				return false, lastErr
			}
			log.Errorf("while reading operation from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, -1, -1, errors.Wrapf(err, "while getting operation by ID: %v", lastErr)
	}
	ret, err := s.toOperationList(operations)
	if err != nil {
		return nil, -1, -1, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, count, totalCount, nil
}

func (s *operations) InsertUpdatingOperation(operation internal.UpdatingOperation) error {
	dto, err := s.updateOperationToDTO(&operation)
	if err != nil {
		return errors.Wrapf(err, "while converting update operation (id: %s)", operation.Operation.ID)
	}

	return s.insert(dto)
}

func (s *operations) GetUpdatingOperationByID(operationID string) (*internal.UpdatingOperation, error) {
	operation, err := s.getByID(operationID)
	if err != nil {
		return nil, errors.Wrap(err, "while getting operation by ID")
	}

	ret, err := s.toUpdateOperation(operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

func (s *operations) UpdateUpdatingOperation(operation internal.UpdatingOperation) (*internal.UpdatingOperation, error) {
	session := s.NewWriteSession()
	operation.UpdatedAt = time.Now()
	dto, err := s.updateOperationToDTO(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting Operation to DTO")
	}

	var lastErr error
	_ = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = session.UpdateOperation(dto)
		if lastErr != nil && dberr.IsNotFound(lastErr) {
			_, lastErr = s.NewReadSession().GetOperationByID(operation.Operation.ID)
			if lastErr != nil {
				log.Errorf("while getting operation: %v", lastErr)
				return false, nil
			}

			// the operation exists but the version is different
			lastErr = dberr.Conflict("operation update conflict, operation ID: %s", operation.Operation.ID)
			log.Warn(lastErr.Error())
			return false, lastErr
		}
		return true, nil
	})
	operation.Version = operation.Version + 1
	return &operation, lastErr
}

// ListUpdatingOperationsByInstanceID Lists all update operations for the given instance
func (s *operations) ListUpdatingOperationsByInstanceID(instanceID string) ([]internal.UpdatingOperation, error) {
	session := s.NewReadSession()
	operations := []dbmodel.OperationDTO{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operations, lastErr = session.GetOperationsByTypeAndInstanceID(instanceID, internal.OperationTypeUpdate)
		if lastErr != nil {
			log.Errorf("while reading operation from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	ret, err := s.toUpdateOperationList(operations)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// InsertUpgradeClusterOperation insert new UpgradeClusterOperation to storage
func (s *operations) InsertUpgradeClusterOperation(operation internal.UpgradeClusterOperation) error {
	dto, err := s.upgradeClusterOperationToDTO(&operation)
	if err != nil {
		return errors.Wrapf(err, "while converting upgrade cluser operation (id: %s)", operation.Operation.ID)
	}

	return s.insert(dto)
}

// UpdateUpgradeClusterOperation updates UpgradeClusterOperation, fails if not exists or optimistic locking failure occurs.
func (s *operations) UpdateUpgradeClusterOperation(operation internal.UpgradeClusterOperation) (*internal.UpgradeClusterOperation, error) {
	session := s.NewWriteSession()
	operation.UpdatedAt = time.Now()
	dto, err := s.upgradeClusterOperationToDTO(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting Operation to DTO")
	}

	var lastErr error
	_ = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = session.UpdateOperation(dto)
		if lastErr != nil && dberr.IsNotFound(lastErr) {
			_, lastErr = s.NewReadSession().GetOperationByID(operation.Operation.ID)
			if lastErr != nil {
				log.Errorf("while getting operation: %v", lastErr)
				return false, nil
			}

			// the operation exists but the version is different
			lastErr = dberr.Conflict("operation update conflict, operation ID: %s", operation.Operation.ID)
			log.Warn(lastErr.Error())
			return false, lastErr
		}
		return true, nil
	})
	operation.Version = operation.Version + 1
	return &operation, lastErr
}

// GetUpgradeClusterOperationByID fetches the UpgradeClusterOperation by given ID, returns error if not found
func (s *operations) GetUpgradeClusterOperationByID(operationID string) (*internal.UpgradeClusterOperation, error) {
	operation, err := s.getByID(operationID)
	if err != nil {
		return nil, errors.Wrap(err, "while getting operation by ID")
	}
	ret, err := s.toUpgradeClusterOperation(operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// ListUpgradeClusterOperationsByInstanceID Lists all upgrade cluster operations for the given instance
func (s *operations) ListUpgradeClusterOperationsByInstanceID(instanceID string) ([]internal.UpgradeClusterOperation, error) {
	session := s.NewReadSession()
	operations := []dbmodel.OperationDTO{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operations, lastErr = session.GetOperationsByTypeAndInstanceID(instanceID, internal.OperationTypeUpgradeCluster)
		if lastErr != nil {
			log.Errorf("while reading operation from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	ret, err := s.toUpgradeClusterOperationList(operations)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// ListUpgradeClusterOperationsByOrchestrationID Lists upgrade cluster operations for the given orchestration, according to filter(s) and/or pagination
func (s *operations) ListUpgradeClusterOperationsByOrchestrationID(orchestrationID string, filter dbmodel.OperationFilter) ([]internal.UpgradeClusterOperation, int, int, error) {
	session := s.NewReadSession()
	var (
		operations        = make([]dbmodel.OperationDTO, 0)
		lastErr           error
		count, totalCount int
	)
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operations, count, totalCount, lastErr = session.ListOperationsByOrchestrationID(orchestrationID, filter)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("Operations for orchestration ID %s not exist", orchestrationID)
				return false, lastErr
			}
			log.Errorf("while reading operation from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, -1, -1, errors.Wrapf(err, "while getting operation by ID: %v", lastErr)
	}
	ret, err := s.toUpgradeClusterOperationList(operations)
	if err != nil {
		return nil, -1, -1, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, count, totalCount, nil
}

func (s *operations) operationToDB(op internal.Operation) (dbmodel.OperationDTO, error) {
	err := s.cipher.EncryptSMCreds(&op.ProvisioningParameters)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrap(err, "while encrypting basic auth")
	}
	pp, err := json.Marshal(op.ProvisioningParameters)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrap(err, "while marshal provisioning parameters")
	}

	stages := []string{}
	for s, _ := range op.FinishedStages {
		stages = append(stages, s)
	}
	return dbmodel.OperationDTO{
		ID:                     op.ID,
		Type:                   op.Type,
		TargetOperationID:      op.ProvisionerOperationID,
		State:                  string(op.State),
		Description:            op.Description,
		UpdatedAt:              op.UpdatedAt,
		CreatedAt:              op.CreatedAt,
		Version:                op.Version,
		InstanceID:             op.InstanceID,
		OrchestrationID:        storage.StringToSQLNullString(op.OrchestrationID),
		ProvisioningParameters: storage.StringToSQLNullString(string(pp)),
		FinishedStages:         storage.StringToSQLNullString(strings.Join(stages, ",")),
	}, nil
}

func (s *operations) toOperation(op *dbmodel.OperationDTO, instanceDetails internal.InstanceDetails) (internal.Operation, error) {
	pp := internal.ProvisioningParameters{}
	if op.ProvisioningParameters.Valid {
		err := json.Unmarshal([]byte(op.ProvisioningParameters.String), &pp)
		if err != nil {
			return internal.Operation{}, errors.Wrap(err, "while unmarshal provisioning parameters")
		}
	}
	err := s.cipher.DecryptSMCreds(&pp)
	if err != nil {
		return internal.Operation{}, errors.Wrap(err, "while decrypting basic auth")
	}

	stages := make(map[string]struct{})
	for _, s := range strings.Split(storage.SQLNullStringToString(op.FinishedStages), ",") {
		stages[s] = struct{}{}
	}
	return internal.Operation{
		ID:                     op.ID,
		CreatedAt:              op.CreatedAt,
		UpdatedAt:              op.UpdatedAt,
		Type:                   op.Type,
		ProvisionerOperationID: op.TargetOperationID,
		State:                  domain.LastOperationState(op.State),
		InstanceID:             op.InstanceID,
		Description:            op.Description,
		Version:                op.Version,
		OrchestrationID:        storage.SQLNullStringToString(op.OrchestrationID),
		ProvisioningParameters: pp,
		InstanceDetails:        instanceDetails,
		FinishedStages:         stages,
	}, nil
}

func (s *operations) toOperations(op []dbmodel.OperationDTO) ([]internal.Operation, error) {
	operations := make([]internal.Operation, 0)
	for _, o := range op {
		operation := internal.Operation{}
		err := json.Unmarshal([]byte(o.Data), &operation)
		if err != nil {
			return nil, errors.New("unable to unmarshall provisioning data")
		}
		operation, err = s.toOperation(&o, operation.InstanceDetails)
		if err != nil {
			return nil, err
		}
		operations = append(operations, operation)
	}
	return operations, nil
}

func (s *operations) toProvisioningOperation(op *dbmodel.OperationDTO) (*internal.ProvisioningOperation, error) {
	if op.Type != internal.OperationTypeProvision {
		return nil, errors.New(fmt.Sprintf("expected operation type Provisioning, but was %s", op.Type))
	}
	var operation internal.ProvisioningOperation
	var err error
	err = json.Unmarshal([]byte(op.Data), &operation)
	if err != nil {
		return nil, errors.New("unable to unmarshall provisioning data")
	}
	operation.Operation, err = s.toOperation(op, operation.InstanceDetails)
	if err != nil {
		return nil, err
	}
	return &operation, nil
}

func (s *operations) toProvisioningOperationList(ops []dbmodel.OperationDTO) ([]internal.ProvisioningOperation, error) {
	result := make([]internal.ProvisioningOperation, 0)

	for _, op := range ops {
		o, err := s.toProvisioningOperation(&op)
		if err != nil {
			return nil, errors.Wrap(err, "while converting to upgrade kyma operation")
		}
		result = append(result, *o)
	}

	return result, nil
}

func (s *operations) toDeprovisioningOperationList(ops []dbmodel.OperationDTO) ([]internal.DeprovisioningOperation, error) {
	result := make([]internal.DeprovisioningOperation, 0)

	for _, op := range ops {
		o, err := s.toDeprovisioningOperation(&op)
		if err != nil {
			return nil, errors.Wrap(err, "while converting to upgrade kyma operation")
		}
		result = append(result, *o)
	}

	return result, nil
}

func (s *operations) operationToDTO(op *internal.Operation) (dbmodel.OperationDTO, error) {
	serialized, err := json.Marshal(op)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrapf(err, "while serializing operation data %v", op)
	}

	ret, err := s.operationToDB(*op)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrapf(err, "while converting to operationDB %v", op)
	}

	ret.Data = string(serialized)
	ret.Type = internal.OperationTypeProvision
	return ret, nil
}

func (s *operations) provisioningOperationToDTO(op *internal.ProvisioningOperation) (dbmodel.OperationDTO, error) {
	serialized, err := json.Marshal(op)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrapf(err, "while serializing provisioning data %v", op)
	}

	ret, err := s.operationToDB(op.Operation)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrapf(err, "while converting to operationDB %v", op)
	}
	ret.Data = string(serialized)
	ret.Type = internal.OperationTypeProvision
	return ret, nil
}

func (s *operations) toDeprovisioningOperation(op *dbmodel.OperationDTO) (*internal.DeprovisioningOperation, error) {
	if op.Type != internal.OperationTypeDeprovision {
		return nil, errors.New(fmt.Sprintf("expected operation type Provisioning, but was %s", op.Type))
	}
	var operation internal.DeprovisioningOperation
	var err error
	err = json.Unmarshal([]byte(op.Data), &operation)
	if err != nil {
		return nil, errors.New("unable to unmarshall provisioning data")
	}
	operation.Operation, err = s.toOperation(op, operation.InstanceDetails)
	if err != nil {
		return nil, err
	}

	return &operation, nil
}

func (s *operations) deprovisioningOperationToDTO(op *internal.DeprovisioningOperation) (dbmodel.OperationDTO, error) {
	serialized, err := json.Marshal(op)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrapf(err, "while serializing deprovisioning data %v", op)
	}

	ret, err := s.operationToDB(op.Operation)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrapf(err, "while converting to operationDB %v", op)
	}
	ret.Data = string(serialized)
	ret.Type = internal.OperationTypeDeprovision
	return ret, nil
}

func (s *operations) toUpgradeKymaOperation(op *dbmodel.OperationDTO) (*internal.UpgradeKymaOperation, error) {
	if op.Type != internal.OperationTypeUpgradeKyma {
		return nil, errors.New(fmt.Sprintf("expected operation type Upgrade Kyma, but was %s", op.Type))
	}
	var operation internal.UpgradeKymaOperation
	var err error
	err = json.Unmarshal([]byte(op.Data), &operation)
	if err != nil {
		return nil, errors.New("unable to unmarshall provisioning data")
	}
	operation.Operation, err = s.toOperation(op, operation.InstanceDetails)
	if err != nil {
		return nil, err
	}
	operation.RuntimeOperation.ID = op.ID
	if op.OrchestrationID.Valid {
		operation.OrchestrationID = op.OrchestrationID.String
	}

	return &operation, nil
}

func (s *operations) toOperationList(ops []dbmodel.OperationDTO) ([]internal.Operation, error) {
	result := make([]internal.Operation, 0)

	for _, op := range ops {

		var operation internal.Operation
		var err error
		err = json.Unmarshal([]byte(op.Data), &operation)
		if err != nil {
			return nil, errors.New("unable to unmarshall provisioning data")
		}

		o, err := s.toOperation(&op, operation.InstanceDetails)
		if err != nil {
			return nil, errors.Wrap(err, "while converting to upgrade kyma operation")
		}
		result = append(result, o)
	}

	return result, nil
}

func (s *operations) toUpgradeKymaOperationList(ops []dbmodel.OperationDTO) ([]internal.UpgradeKymaOperation, error) {
	result := make([]internal.UpgradeKymaOperation, 0)

	for _, op := range ops {
		o, err := s.toUpgradeKymaOperation(&op)
		if err != nil {
			return nil, errors.Wrap(err, "while converting to upgrade kyma operation")
		}
		result = append(result, *o)
	}

	return result, nil
}

func (s *operations) upgradeKymaOperationToDTO(op *internal.UpgradeKymaOperation) (dbmodel.OperationDTO, error) {
	serialized, err := json.Marshal(op)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrapf(err, "while serializing provisioning data %v", op)
	}

	ret, err := s.operationToDB(op.Operation)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrapf(err, "while converting to operationDB %v", op)
	}
	ret.Data = string(serialized)
	ret.Type = internal.OperationTypeUpgradeKyma
	ret.OrchestrationID = storage.StringToSQLNullString(op.OrchestrationID)
	return ret, nil
}

func (s *operations) toUpgradeClusterOperation(op *dbmodel.OperationDTO) (*internal.UpgradeClusterOperation, error) {
	if op.Type != internal.OperationTypeUpgradeCluster {
		return nil, errors.New(fmt.Sprintf("expected operation type upgradeCluster, but was %s", op.Type))
	}
	var operation internal.UpgradeClusterOperation
	var err error
	err = json.Unmarshal([]byte(op.Data), &operation)
	if err != nil {
		return nil, errors.New("unable to unmarshall provisioning data")
	}
	operation.Operation, err = s.toOperation(op, operation.InstanceDetails)
	if err != nil {
		return nil, err
	}
	operation.RuntimeOperation.ID = op.ID
	if op.OrchestrationID.Valid {
		operation.OrchestrationID = op.OrchestrationID.String
	}

	return &operation, nil
}

func (s *operations) toUpgradeClusterOperationList(ops []dbmodel.OperationDTO) ([]internal.UpgradeClusterOperation, error) {
	result := make([]internal.UpgradeClusterOperation, 0)

	for _, op := range ops {
		o, err := s.toUpgradeClusterOperation(&op)
		if err != nil {
			return nil, errors.Wrap(err, "while converting to upgrade cluster operation")
		}
		result = append(result, *o)
	}

	return result, nil
}

func (s *operations) upgradeClusterOperationToDTO(op *internal.UpgradeClusterOperation) (dbmodel.OperationDTO, error) {
	serialized, err := json.Marshal(op)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrapf(err, "while serializing upgradeCluster data %v", op)
	}

	ret, err := s.operationToDB(op.Operation)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrapf(err, "while converting to operationDB %v", op)
	}
	ret.Data = string(serialized)
	ret.Type = internal.OperationTypeUpgradeCluster
	ret.OrchestrationID = storage.StringToSQLNullString(op.OrchestrationID)
	return ret, nil
}

func (s *operations) updateOperationToDTO(op *internal.UpdatingOperation) (dbmodel.OperationDTO, error) {
	serialized, err := json.Marshal(op)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrapf(err, "while serializing update data %v", op)
	}

	ret, err := s.operationToDB(op.Operation)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrapf(err, "while converting to operationDB %v", op)
	}
	ret.Data = string(serialized)
	ret.Type = internal.OperationTypeUpdate
	ret.OrchestrationID = storage.StringToSQLNullString(op.OrchestrationID)
	return ret, nil
}

func (s *operations) toUpdateOperation(op *dbmodel.OperationDTO) (*internal.UpdatingOperation, error) {
	if op.Type != internal.OperationTypeUpdate {
		return nil, errors.New(fmt.Sprintf("expected operation type update, but was %s", op.Type))
	}
	var operation internal.UpdatingOperation
	var err error
	err = json.Unmarshal([]byte(op.Data), &operation)
	if err != nil {
		return nil, errors.New("unable to unmarshall provisioning data")
	}
	operation.Operation, err = s.toOperation(op, operation.InstanceDetails)
	if err != nil {
		return nil, err
	}

	return &operation, nil
}

func (s *operations) toUpdateOperationList(ops []dbmodel.OperationDTO) ([]internal.UpdatingOperation, error) {
	result := make([]internal.UpdatingOperation, 0)

	for _, op := range ops {
		o, err := s.toUpdateOperation(&op)
		if err != nil {
			return nil, errors.Wrap(err, "while converting to upgrade cluster operation")
		}
		result = append(result, *o)
	}

	return result, nil
}

func (s *operations) getByID(id string) (*dbmodel.OperationDTO, error) {
	var lastErr dberr.Error
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}

	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetOperationByID(id)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("Operation with id %s not exist", id)
				return false, lastErr
			}
			log.Errorf("while reading operation from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return &operation, nil
}

func (s *operations) insert(dto dbmodel.OperationDTO) error {
	session := s.NewWriteSession()
	var lastErr error
	_ = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = session.InsertOperation(dto)
		if lastErr != nil {
			log.Errorf("while insert operation: %v", lastErr)
			return false, nil
		}
		// TODO: insert link to orchestration
		return true, nil
	})
	return lastErr
}

func (s *operations) getByInstanceID(id string) (*dbmodel.OperationDTO, error) {
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetOperationByInstanceID(id)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("operation does not exist")
				return false, lastErr
			}
			log.Errorf("while reading operation from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})

	return &operation, err
}

func (s *operations) getByTypeAndInstanceID(id string, opType internal.OperationType) (*dbmodel.OperationDTO, error) {
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetOperationByTypeAndInstanceID(id, opType)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("operation does not exist")
				return false, lastErr
			}
			log.Errorf("while reading operation from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})

	return &operation, err
}

func (s *operations) update(operation dbmodel.OperationDTO) error {
	session := s.NewWriteSession()

	var lastErr error
	_ = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = session.UpdateOperation(operation)
		if lastErr != nil && dberr.IsNotFound(lastErr) {
			_, lastErr = s.NewReadSession().GetOperationByID(operation.ID)
			if lastErr != nil {
				log.Errorf("while getting operation: %v", lastErr)
				return false, nil
			}

			// the operation exists but the version is different
			lastErr = dberr.Conflict("operation update conflict, operation ID: %s", operation.ID)
			log.Warn(lastErr.Error())
			return false, lastErr
		}
		return true, nil
	})
	return lastErr
}

func (s *operations) listOperationsByInstanceIdAndType(instanceId string, operationType internal.OperationType) ([]dbmodel.OperationDTO, error) {
	session := s.NewReadSession()
	operations := []dbmodel.OperationDTO{}
	var lastErr dberr.Error

	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operations, lastErr = session.GetOperationsByTypeAndInstanceID(instanceId, operationType)
		if lastErr != nil {
			log.Errorf("while reading operation from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	return operations, lastErr
}

func (s *operations) listOperationsByType(operationType internal.OperationType) ([]dbmodel.OperationDTO, error) {
	session := s.NewReadSession()
	operations := []dbmodel.OperationDTO{}
	var lastErr dberr.Error

	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operations, lastErr = session.ListOperationsByType(operationType)
		if lastErr != nil {
			log.Errorf("while reading operation from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	return operations, lastErr
}

func (s *operations) listOperationsByInstanceId(instanceId string) ([]dbmodel.OperationDTO, error) {
	session := s.NewReadSession()
	operations := []dbmodel.OperationDTO{}
	var lastErr dberr.Error

	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operations, lastErr = session.GetOperationsByInstanceID(instanceId)
		if lastErr != nil {
			log.Errorf("while reading operation from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	return operations, lastErr
}
