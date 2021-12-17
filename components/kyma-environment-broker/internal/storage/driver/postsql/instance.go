package postsql

import (
	"encoding/json"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/predicate"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Instance struct {
	postsql.Factory
	operations *operations
	cipher     Cipher
}

func NewInstance(sess postsql.Factory, operations *operations, cipher Cipher) *Instance {
	return &Instance{
		Factory:    sess,
		operations: operations,
		cipher:     cipher,
	}
}

func (s *Instance) InsertWithoutEncryption(instance internal.Instance) error {
	_, err := s.GetByID(instance.InstanceID)
	if err == nil {
		return dberr.AlreadyExists("instance with id %s already exist", instance.InstanceID)
	}
	params, err := json.Marshal(instance.Parameters)
	if err != nil {
		return errors.Wrap(err, "while marshaling parameters")
	}
	dto := dbmodel.InstanceDTO{
		InstanceID:             instance.InstanceID,
		RuntimeID:              instance.RuntimeID,
		GlobalAccountID:        instance.GlobalAccountID,
		SubAccountID:           instance.SubAccountID,
		ServiceID:              instance.ServiceID,
		ServiceName:            instance.ServiceName,
		ServicePlanID:          instance.ServicePlanID,
		ServicePlanName:        instance.ServicePlanName,
		DashboardURL:           instance.DashboardURL,
		ProvisioningParameters: string(params),
		ProviderRegion:         instance.ProviderRegion,
		CreatedAt:              instance.CreatedAt,
		UpdatedAt:              instance.UpdatedAt,
		DeletedAt:              instance.DeletedAt,
		Version:                instance.Version,
		Provider:               string(instance.Provider),
	}

	sess := s.NewWriteSession()
	return wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		err := sess.InsertInstance(dto)
		if err != nil {
			log.Errorf("while saving instance ID %s: %v", instance.InstanceID, err)
			return false, nil
		}
		return true, nil
	})
}

func (s *Instance) ListWithoutDecryption(filter dbmodel.InstanceFilter) ([]internal.Instance, int, int, error) {
	dtos, count, totalCount, err := s.NewReadSession().ListInstances(filter)
	if err != nil {
		return []internal.Instance{}, 0, 0, err
	}
	var instances []internal.Instance
	for _, dto := range dtos {
		var params internal.ProvisioningParameters
		err := json.Unmarshal([]byte(dto.ProvisioningParameters), &params)
		if err != nil {
			return nil, 0, 0, errors.Wrap(err, "while unmarshal parameters")
		}
		instance := internal.Instance{
			InstanceID:      dto.InstanceID,
			RuntimeID:       dto.RuntimeID,
			GlobalAccountID: dto.GlobalAccountID,
			SubAccountID:    dto.SubAccountID,
			ServiceID:       dto.ServiceID,
			ServiceName:     dto.ServiceName,
			ServicePlanID:   dto.ServicePlanID,
			ServicePlanName: dto.ServicePlanName,
			DashboardURL:    dto.DashboardURL,
			Parameters:      params,
			ProviderRegion:  dto.ProviderRegion,
			CreatedAt:       dto.CreatedAt,
			UpdatedAt:       dto.UpdatedAt,
			DeletedAt:       dto.DeletedAt,
			Version:         dto.Version,
			Provider:        internal.CloudProvider(dto.Provider),
		}
		instances = append(instances, instance)
	}
	return instances, count, totalCount, err
}

func (s *Instance) UpdateWithoutEncryption(instance internal.Instance) (*internal.Instance, error) {
	sess := s.NewWriteSession()
	params, err := json.Marshal(instance.Parameters)
	if err != nil {
		return nil, errors.Wrap(err, "while marshaling parameters")
	}
	dto := dbmodel.InstanceDTO{
		InstanceID:             instance.InstanceID,
		RuntimeID:              instance.RuntimeID,
		GlobalAccountID:        instance.GlobalAccountID,
		SubAccountID:           instance.SubAccountID,
		ServiceID:              instance.ServiceID,
		ServiceName:            instance.ServiceName,
		ServicePlanID:          instance.ServicePlanID,
		ServicePlanName:        instance.ServicePlanName,
		DashboardURL:           instance.DashboardURL,
		ProvisioningParameters: string(params),
		ProviderRegion:         instance.ProviderRegion,
		CreatedAt:              instance.CreatedAt,
		UpdatedAt:              instance.UpdatedAt,
		DeletedAt:              instance.DeletedAt,
		Version:                instance.Version,
		Provider:               string(instance.Provider),
	}
	var lastErr dberr.Error
	err = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = sess.UpdateInstance(dto)

		switch {
		case dberr.IsNotFound(lastErr):
			_, lastErr = s.NewReadSession().GetInstanceByID(instance.InstanceID)
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("Instance with id %s not exist", instance.InstanceID)
			}
			if lastErr != nil {
				log.Warn(errors.Wrapf(lastErr, "while getting Operation").Error())
				return false, nil
			}

			// the operation exists but the version is different
			lastErr = dberr.Conflict("operation update conflict, operation ID: %s", instance.InstanceID)
			return false, lastErr
		case lastErr != nil:
			log.Errorf("while updating instance ID %s: %v", instance.InstanceID, lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	instance.Version = instance.Version + 1
	return &instance, nil
}

func (s *Instance) FindAllJoinedWithOperations(prct ...predicate.Predicate) ([]internal.InstanceWithOperation, error) {
	sess := s.NewReadSession()
	var (
		instances []dbmodel.InstanceWithOperationDTO
		lastErr   dberr.Error
	)
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		instances, lastErr = sess.FindAllInstancesJoinedWithOperation(prct...)
		if lastErr != nil {
			log.Errorf("while fetching all instances: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}

	var result []internal.InstanceWithOperation
	for _, dto := range instances {
		inst, err := s.toInstance(dto.InstanceDTO)
		if err != nil {
			return nil, err
		}

		var isSuspensionOp bool

		switch internal.OperationType(dto.Type.String) {
		case internal.OperationTypeProvision:
			isSuspensionOp = false
		case internal.OperationTypeDeprovision:
			deprovOp, err := s.toDeprovisioningOp(&dto)
			if err != nil {
				log.Errorf("while unmarshalling DTO deprovisioning operation data: %v", err)
			}
			isSuspensionOp = deprovOp.Temporary
		}

		result = append(result, internal.InstanceWithOperation{
			Instance:       inst,
			Type:           dto.Type,
			State:          dto.State,
			Description:    dto.Description,
			OpCreatedAt:    dto.OperationCreatedAt.Time,
			IsSuspensionOp: isSuspensionOp,
		})
	}

	return result, nil
}

func (s *Instance) toProvisioningOp(dto *dbmodel.InstanceWithOperationDTO) (*internal.ProvisioningOperation, error) {
	var provOp internal.ProvisioningOperation
	err := json.Unmarshal([]byte(dto.Data.String), &provOp)
	if err != nil {
		return nil, errors.New("unable to unmarshall provisioning data")
	}

	return &provOp, nil
}

func (s *Instance) toDeprovisioningOp(dto *dbmodel.InstanceWithOperationDTO) (*internal.DeprovisioningOperation, error) {
	var deprovOp internal.DeprovisioningOperation
	err := json.Unmarshal([]byte(dto.Data.String), &deprovOp)
	if err != nil {
		return nil, errors.New("unable to unmarshall deprovisioning data")
	}

	return &deprovOp, nil
}

func (s *Instance) FindAllInstancesForRuntimes(runtimeIdList []string) ([]internal.Instance, error) {
	sess := s.NewReadSession()
	var instances []dbmodel.InstanceDTO
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		instances, lastErr = sess.FindAllInstancesForRuntimes(runtimeIdList)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("Instances with runtime IDs from list '%+q' not exist", runtimeIdList)
			}
			log.Errorf("while getting instances from runtime ID list '%+q': %v", runtimeIdList, lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}

	var result []internal.Instance
	for _, dto := range instances {
		inst, err := s.toInstance(dto)
		if err != nil {
			return []internal.Instance{}, err
		}
		result = append(result, inst)
	}

	return result, nil
}

func (s *Instance) FindAllInstancesForSubAccounts(subAccountslist []string) ([]internal.Instance, error) {
	sess := s.NewReadSession()
	var (
		instances []dbmodel.InstanceDTO
		lastErr   dberr.Error
	)
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		instances, lastErr = sess.FindAllInstancesForSubAccounts(subAccountslist)
		if lastErr != nil {
			log.Errorf("while fetching instances by subaccount list: %v", lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}

	var result []internal.Instance
	for _, dto := range instances {
		inst, err := s.toInstance(dto)
		if err != nil {
			return []internal.Instance{}, err
		}
		result = append(result, inst)
	}

	return result, nil
}

func (s *Instance) GetNumberOfInstancesForGlobalAccountID(globalAccountID string) (int, error) {
	sess := s.NewReadSession()
	var result int
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		count, err := sess.GetNumberOfInstancesForGlobalAccountID(globalAccountID)
		result = count
		return err == nil, nil
	})
	return result, err
}

// TODO: Wrap retries in single method WithRetries
func (s *Instance) GetByID(instanceID string) (*internal.Instance, error) {
	sess := s.NewReadSession()
	instanceDTO := dbmodel.InstanceDTO{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		instanceDTO, lastErr = sess.GetInstanceByID(instanceID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("Instance with id %s not exist", instanceID)
			}
			log.Errorf("while getting instanceDTO by ID %s: %v", instanceID, lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	instance, err := s.toInstance(instanceDTO)
	if err != nil {
		return nil, err
	}

	lastOp, err := s.operations.GetLastOperation(instanceID)
	if err != nil {
		if dberr.IsNotFound(err) {
			return &instance, nil
		}
		return nil, err
	}
	instance.InstanceDetails = lastOp.InstanceDetails
	return &instance, nil
}

func (s *Instance) toInstance(dto dbmodel.InstanceDTO) (internal.Instance, error) {
	var params internal.ProvisioningParameters
	err := json.Unmarshal([]byte(dto.ProvisioningParameters), &params)
	if err != nil {
		return internal.Instance{}, errors.Wrap(err, "while unmarshal parameters")
	}
	err = s.cipher.DecryptSMCreds(&params)
	if err != nil {
		return internal.Instance{}, errors.Wrap(err, "while decrypting parameters")
	}
	return internal.Instance{
		InstanceID:      dto.InstanceID,
		RuntimeID:       dto.RuntimeID,
		GlobalAccountID: dto.GlobalAccountID,
		SubAccountID:    dto.SubAccountID,
		ServiceID:       dto.ServiceID,
		ServiceName:     dto.ServiceName,
		ServicePlanID:   dto.ServicePlanID,
		ServicePlanName: dto.ServicePlanName,
		DashboardURL:    dto.DashboardURL,
		Parameters:      params,
		ProviderRegion:  dto.ProviderRegion,
		CreatedAt:       dto.CreatedAt,
		UpdatedAt:       dto.UpdatedAt,
		DeletedAt:       dto.DeletedAt,
		Version:         dto.Version,
		Provider:        internal.CloudProvider(dto.Provider),
	}, nil
}

func (s *Instance) Insert(instance internal.Instance) error {
	_, err := s.GetByID(instance.InstanceID)
	if err == nil {
		return dberr.AlreadyExists("instance with id %s already exist", instance.InstanceID)
	}

	dto, err := s.toInstanceDTO(instance)
	if err != nil {
		return err
	}

	sess := s.NewWriteSession()
	return wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		err := sess.InsertInstance(dto)
		if err != nil {
			log.Errorf("while saving instance ID %s: %v", instance.InstanceID, err)
			return false, nil
		}
		return true, nil
	})
}

func (s *Instance) Update(instance internal.Instance) (*internal.Instance, error) {
	sess := s.NewWriteSession()
	dto, err := s.toInstanceDTO(instance)
	if err != nil {
		return nil, err
	}
	var lastErr dberr.Error
	err = wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		lastErr = sess.UpdateInstance(dto)

		switch {
		case dberr.IsNotFound(lastErr):
			_, lastErr = s.NewReadSession().GetInstanceByID(instance.InstanceID)
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("Instance with id %s not exist", instance.InstanceID)
			}
			if lastErr != nil {
				log.Warn(errors.Wrapf(lastErr, "while getting Operation").Error())
				return false, nil
			}

			// the operation exists but the version is different
			lastErr = dberr.Conflict("operation update conflict, operation ID: %s", instance.InstanceID)
			return false, lastErr
		case lastErr != nil:
			log.Errorf("while updating instance ID %s: %v", instance.InstanceID, lastErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	instance.Version = instance.Version + 1
	return &instance, nil
}

func (s *Instance) toInstanceDTO(instance internal.Instance) (dbmodel.InstanceDTO, error) {
	err := s.cipher.EncryptSMCreds(&instance.Parameters)
	if err != nil {
		return dbmodel.InstanceDTO{}, errors.Wrap(err, "while encrypting parameters")
	}
	params, err := json.Marshal(instance.Parameters)
	if err != nil {
		return dbmodel.InstanceDTO{}, errors.Wrap(err, "while marshaling parameters")
	}
	return dbmodel.InstanceDTO{
		InstanceID:             instance.InstanceID,
		RuntimeID:              instance.RuntimeID,
		GlobalAccountID:        instance.GlobalAccountID,
		SubAccountID:           instance.SubAccountID,
		ServiceID:              instance.ServiceID,
		ServiceName:            instance.ServiceName,
		ServicePlanID:          instance.ServicePlanID,
		ServicePlanName:        instance.ServicePlanName,
		DashboardURL:           instance.DashboardURL,
		ProvisioningParameters: string(params),
		ProviderRegion:         instance.ProviderRegion,
		CreatedAt:              instance.CreatedAt,
		UpdatedAt:              instance.UpdatedAt,
		DeletedAt:              instance.DeletedAt,
		Version:                instance.Version,
		Provider:               string(instance.Provider),
	}, nil
}

func (s *Instance) Delete(instanceID string) error {
	sess := s.NewWriteSession()
	return sess.DeleteInstance(instanceID)
}

func (s *Instance) GetInstanceStats() (internal.InstanceStats, error) {
	entries, err := s.NewReadSession().GetInstanceStats()
	if err != nil {
		return internal.InstanceStats{}, err
	}

	result := internal.InstanceStats{
		PerGlobalAccountID: make(map[string]int),
	}
	for _, e := range entries {
		result.PerGlobalAccountID[e.GlobalAccountID] = e.Total
		result.TotalNumberOfInstances = result.TotalNumberOfInstances + e.Total
	}
	return result, nil
}

func (s *Instance) List(filter dbmodel.InstanceFilter) ([]internal.Instance, int, int, error) {
	dtos, count, totalCount, err := s.NewReadSession().ListInstances(filter)
	if err != nil {
		return []internal.Instance{}, 0, 0, err
	}
	var instances []internal.Instance
	for _, dto := range dtos {
		instance, err := s.toInstance(dto)
		if err != nil {
			return []internal.Instance{}, 0, 0, err
		}
		instances = append(instances, instance)
	}
	return instances, count, totalCount, err
}
