package postsql

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/storage"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/pivotal-cf/brokerapi/v7/domain"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/pkg/errors"
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

// used only as memory storage in the unit test
func (s *operations) GetLegacyOperation(operationID string) (*internal.LegacyOperation, error) {
	return nil, nil
}
func (s *operations) InsertLegacyOperation(operation internal.LegacyOperation) error {
	return nil
}

// InsertProvisioningOperation insert new ProvisioningOperation to storage
func (s *operations) InsertProvisioningOperation(operation internal.ProvisioningOperation) error {
	session := s.NewWriteSession()
	dto, err := s.provisioningOperationToDTO(&operation)
	if err != nil {
		return errors.Wrapf(err, "while inserting provisioning operation (id: %s)", operation.ID)
	}
	var lastErr error
	_ = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = session.InsertOperation(dto)
		if lastErr != nil {
			log.Errorf("while inserting operation: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	return lastErr
}

// GetProvisioningOperationByID fetches the ProvisioningOperation by given ID, returns error if not found
func (s *operations) GetProvisioningOperationByID(operationID string) (*internal.ProvisioningOperation, error) {
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}
	var lastErr error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetOperationByID(operationID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("Operation with id %s not exist", operationID)
				return false, lastErr
			}
			log.Errorf("while reading operation from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "while getting operation by ID")
	}
	ret, err := s.toProvisioningOperation(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// GetProvisioningOperationByInstanceID fetches the ProvisioningOperation by given instanceID, returns error if not found
func (s *operations) GetProvisioningOperationByInstanceID(instanceID string) (*internal.ProvisioningOperation, error) {
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetOperationByTypeAndInstanceID(instanceID, dbmodel.OperationTypeProvision)
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
	if err != nil {
		return nil, lastErr
	}
	ret, err := s.toProvisioningOperation(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// UpdateProvisioningOperation updates ProvisioningOperation, fails if not exists or optimistic locking failure occurs.
func (s *operations) UpdateProvisioningOperation(op internal.ProvisioningOperation) (*internal.ProvisioningOperation, error) {
	session := s.NewWriteSession()
	op.UpdatedAt = time.Now()
	dto, err := s.provisioningOperationToDTO(&op)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting Operation to DTO")
	}

	var lastErr error
	_ = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = session.UpdateOperation(dto)
		if lastErr != nil && dberr.IsNotFound(lastErr) {
			_, lastErr = s.NewReadSession().GetOperationByID(op.ID)
			if lastErr != nil {
				log.Errorf("while getting operation: %v", lastErr)
				return false, nil
			}

			// the operation exists but the version is different
			lastErr = dberr.Conflict("operation update conflict, operation ID: %s", op.ID)
			log.Warn(lastErr.Error())
			return false, lastErr
		}
		return true, nil
	})
	op.Version = op.Version + 1
	return &op, lastErr
}

// InsertDeprovisioningOperation insert new DeprovisioningOperation to storage
func (s *operations) InsertDeprovisioningOperation(operation internal.DeprovisioningOperation) error {
	session := s.NewWriteSession()

	dto, err := s.deprovisioningOperationToDTO(&operation)
	if err != nil {
		return errors.Wrapf(err, "while converting Operation to DTO")
	}

	var lastErr error
	_ = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = session.InsertOperation(dto)
		if lastErr != nil {
			log.Errorf("while insert operation: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	return lastErr
}

// GetDeprovisioningOperationByID fetches the DeprovisioningOperation by given ID, returns error if not found
func (s *operations) GetDeprovisioningOperationByID(operationID string) (*internal.DeprovisioningOperation, error) {
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}
	var lastErr error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetOperationByID(operationID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("Operation with id %s not exist", operationID)
				return false, lastErr
			}
			log.Errorf("while reading operation from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "while getting operation by ID")
	}
	ret, err := s.toDeprovisioningOperation(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// GetDeprovisioningOperationByInstanceID fetches the DeprovisioningOperation by given instanceID, returns error if not found
func (s *operations) GetDeprovisioningOperationByInstanceID(instanceID string) (*internal.DeprovisioningOperation, error) {
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetOperationByTypeAndInstanceID(instanceID, dbmodel.OperationTypeDeprovision)
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
	if err != nil {
		return nil, lastErr
	}
	ret, err := s.toDeprovisioningOperation(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// UpdateDeprovisioningOperation updates DeprovisioningOperation, fails if not exists or optimistic locking failure occurs.
func (s *operations) UpdateDeprovisioningOperation(operation internal.DeprovisioningOperation) (*internal.DeprovisioningOperation, error) {
	session := s.NewWriteSession()
	operation.UpdatedAt = time.Now()

	dto, err := s.deprovisioningOperationToDTO(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting Operation to DTO")
	}

	var lastErr error
	_ = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = session.UpdateOperation(dto)
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
	operation.Version = operation.Version + 1
	return &operation, lastErr
}

// InsertUpgradeKymaOperation insert new UpgradeKymaOperation to storage
func (s *operations) InsertUpgradeKymaOperation(operation internal.UpgradeKymaOperation) error {
	session := s.NewWriteSession()
	dto, err := s.upgradeKymaOperationToDTO(&operation)
	if err != nil {
		return errors.Wrapf(err, "while inserting upgrade kyma operation (id: %s)", operation.Operation.ID)
	}
	var lastErr error
	_ = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = session.InsertOperation(dto)
		if lastErr != nil {
			log.Errorf("while insert operation: %v", err)
			return false, nil
		}

		//todo - insert link to orchestration
		return true, nil
	})
	return lastErr
}

// GetUpgradeKymaOperationByID fetches the UpgradeKymaOperation by given ID, returns error if not found
func (s *operations) GetUpgradeKymaOperationByID(operationID string) (*internal.UpgradeKymaOperation, error) {
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}
	var lastErr error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetOperationByID(operationID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("Operation with id %s not exist", operationID)
				return false, lastErr
			}
			log.Errorf("while reading operation from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "while getting operation by ID")
	}
	ret, err := s.toUpgradeKymaOperation(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// GetUpgradeKymaOperationByInstanceID fetches the latest UpgradeKymaOperation by given instanceID, returns error if not found
func (s *operations) GetUpgradeKymaOperationByInstanceID(instanceID string) (*internal.UpgradeKymaOperation, error) {
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetOperationByTypeAndInstanceID(instanceID, dbmodel.OperationTypeUpgradeKyma)
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
	if err != nil {
		return nil, lastErr
	}
	ret, err := s.toUpgradeKymaOperation(&operation)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

func (s *operations) ListUpgradeKymaOperationsByInstanceID(instanceID string) ([]internal.UpgradeKymaOperation, error) {
	session := s.NewReadSession()
	operations := []dbmodel.OperationDTO{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operations, lastErr = session.GetOperationsByTypeAndInstanceID(instanceID, dbmodel.OperationTypeUpgradeKyma)
		if lastErr != nil {
			log.Errorf("while reading operation from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	ret, err := s.toUpgradeKymaOperationList(operations)
	if err != nil {
		return nil, errors.Wrapf(err, "while converting DTO to Operation")
	}

	return ret, nil
}

// UpdateUpgradeKymaOperation updates UpgradeKymaOperation, fails if not exists or optimistic locking failure occurs.
func (s *operations) UpdateUpgradeKymaOperation(operation internal.UpgradeKymaOperation) (*internal.UpgradeKymaOperation, error) {
	session := s.NewWriteSession()
	operation.UpdatedAt = time.Now()
	dto, err := s.upgradeKymaOperationToDTO(&operation)
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
	session := s.NewReadSession()
	operation := dbmodel.OperationDTO{}
	op := internal.Operation{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		operation, lastErr = session.GetOperationByID(operationID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("Operation with id %s not exist", operationID)
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

func (s *operations) GetOperationsInProgressByType(operationType dbmodel.OperationType) ([]internal.Operation, error) {
	session := s.NewReadSession()
	operations := make([]dbmodel.OperationDTO, 0)
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		dto, err := session.GetOperationsInProgressByType(operationType)
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
		switch dbmodel.OperationType(e.Type) {
		case dbmodel.OperationTypeProvision:
			result[e.PlanID].Provisioning[domain.LastOperationState(e.State)] += 1
		case dbmodel.OperationTypeDeprovision:
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
	result := make(map[string]int)
	for _, entry := range entries {
		result[entry.State] += 1
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

func (s *operations) ListOperationsParameters() (map[string]internal.ProvisioningParameters, error) {
	session := s.NewReadSession()

	var (
		lastErr    error
		parameters = make(map[string]internal.ProvisioningParameters, 0)
	)

	_ = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		parameters, lastErr = session.ListOperationsParameters()
		if lastErr != nil {
			log.Errorf("while getting operations from the storage: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	return parameters, lastErr
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

func (s *operations) UpdateOperationParameters(operation internal.Operation) (*internal.Operation, error) {
	session := s.NewWriteSession()
	operation.UpdatedAt = time.Now()
	dto, err := s.operationToDB(operation)
	if err != nil {
		return &internal.Operation{}, errors.Wrapf(err, "while converting to operationDB %v", operation)
	}

	var lastErr error
	_ = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = session.UpdateOperationParameters(dto)
		if lastErr != nil && dberr.IsNotFound(lastErr) {
			newOp, lastErr := s.NewReadSession().GetOperationByID(operation.ID)
			if lastErr != nil {
				log.Errorf("while getting operation: %v", lastErr)
				return false, nil
			}

			// the operation exists but the version is different
			dto.Version = newOp.Version
		}
		return true, nil
	})
	operation.Version = operation.Version + 1
	return &operation, lastErr
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

func (s *operations) encryptBasicAuth(pp *internal.ProvisioningParameters) error {
	if pp.ErsContext.ServiceManager == nil {
		return nil
	}
	creds := pp.ErsContext.ServiceManager.Credentials.BasicAuth
	if creds.Username == "" || creds.Password == "" {
		return nil
	}
	username, err := s.cipher.Encrypt([]byte(pp.ErsContext.ServiceManager.Credentials.BasicAuth.Username))
	if err != nil {
		return errors.Wrap(err, "while encrypting username")
	}
	password, err := s.cipher.Encrypt([]byte(pp.ErsContext.ServiceManager.Credentials.BasicAuth.Password))
	if err != nil {
		return errors.Wrap(err, "while encrypting password")
	}

	pp.ErsContext.ServiceManager = &internal.ServiceManagerEntryDTO{
		Credentials: internal.ServiceManagerCredentials{
			BasicAuth: internal.ServiceManagerBasicAuth{
				Username: string(username),
				Password: string(password),
			}},
		URL: pp.ErsContext.ServiceManager.URL,
	}

	return nil
}

func (s *operations) decryptBasicAuth(pp *internal.ProvisioningParameters) error {
	if pp.ErsContext.ServiceManager == nil {
		return nil
	}
	creds := pp.ErsContext.ServiceManager.Credentials.BasicAuth
	if creds.Username == "" || creds.Password == "" {
		return nil
	}
	username, err := s.cipher.Decrypt([]byte(creds.Username))
	if err != nil {
		return errors.Wrap(err, "while decrypting username")
	}
	password, err := s.cipher.Decrypt([]byte(creds.Password))
	if err != nil {
		return errors.Wrap(err, "while decrypting password")
	}

	pp.ErsContext.ServiceManager.Credentials.BasicAuth.Username = string(username)
	pp.ErsContext.ServiceManager.Credentials.BasicAuth.Password = string(password)

	return nil
}

func (s *operations) operationToDB(op internal.Operation) (dbmodel.OperationDTO, error) {
	err := s.encryptBasicAuth(&op.ProvisioningParameters)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrap(err, "while encrypting basic auth")
	}
	pp, err := json.Marshal(op.ProvisioningParameters)
	if err != nil {
		return dbmodel.OperationDTO{}, errors.Wrap(err, "while marshal provisioning parameters")
	}

	return dbmodel.OperationDTO{
		ID:                     op.ID,
		TargetOperationID:      op.ProvisionerOperationID,
		State:                  string(op.State),
		Description:            op.Description,
		UpdatedAt:              op.UpdatedAt,
		CreatedAt:              op.CreatedAt,
		Version:                op.Version,
		InstanceID:             op.InstanceID,
		OrchestrationID:        storage.StringToSQLNullString(op.OrchestrationID),
		ProvisioningParameters: storage.StringToSQLNullString(string(pp)),
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
	err := s.decryptBasicAuth(&pp)
	if err != nil {
		return internal.Operation{}, errors.Wrap(err, "while decrypting basic auth")
	}

	return internal.Operation{
		ID:                     op.ID,
		CreatedAt:              op.CreatedAt,
		UpdatedAt:              op.UpdatedAt,
		ProvisionerOperationID: op.TargetOperationID,
		State:                  domain.LastOperationState(op.State),
		InstanceID:             op.InstanceID,
		Description:            op.Description,
		Version:                op.Version,
		OrchestrationID:        storage.SQLNullStringToString(op.OrchestrationID),
		ProvisioningParameters: pp,
		InstanceDetails:        instanceDetails,
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
	if op.Type != dbmodel.OperationTypeProvision {
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
	ret.Type = dbmodel.OperationTypeProvision
	return ret, nil
}

func (s *operations) toDeprovisioningOperation(op *dbmodel.OperationDTO) (*internal.DeprovisioningOperation, error) {
	if op.Type != dbmodel.OperationTypeDeprovision {
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
	ret.Type = dbmodel.OperationTypeDeprovision
	return ret, nil
}

func (s *operations) toUpgradeKymaOperation(op *dbmodel.OperationDTO) (*internal.UpgradeKymaOperation, error) {
	if op.Type != dbmodel.OperationTypeUpgradeKyma {
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
	ret.Type = dbmodel.OperationTypeUpgradeKyma
	ret.OrchestrationID = storage.StringToSQLNullString(op.OrchestrationID)
	return ret, nil
}
