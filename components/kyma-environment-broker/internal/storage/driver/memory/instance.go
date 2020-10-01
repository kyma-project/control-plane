package memory

import (
	"database/sql"
	"sort"
	"sync"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/pagination"

	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/predicate"
)

type Instance struct {
	mu                sync.Mutex
	instances         map[string]internal.Instance
	operationsStorage *operations
}

func NewInstance(operations *operations) *Instance {
	return &Instance{
		instances:         make(map[string]internal.Instance, 0),
		operationsStorage: operations,
	}
}
func (s *Instance) FindAllJoinedWithOperations(prct ...predicate.Predicate) ([]internal.InstanceWithOperation, error) {
	var instances []internal.InstanceWithOperation

	// simulate left join without grouping on column
	for id, v := range s.instances {
		dOp, dErr := s.operationsStorage.GetDeprovisioningOperationByInstanceID(id)
		if dErr != nil && !dberr.IsNotFound(dErr) {
			return nil, dErr
		}
		pOp, pErr := s.operationsStorage.GetProvisioningOperationByInstanceID(id)
		if pErr != nil && !dberr.IsNotFound(pErr) {
			return nil, pErr
		}
		uOp, uErr := s.operationsStorage.GetUpgradeKymaOperationByInstanceID(id)
		if uErr != nil && !dberr.IsNotFound(uErr) {
			return nil, uErr
		}

		if !dberr.IsNotFound(dErr) {
			instances = append(instances, internal.InstanceWithOperation{
				Instance:    v,
				Type:        sql.NullString{String: string(dbmodel.OperationTypeDeprovision), Valid: true},
				State:       sql.NullString{String: string(dOp.State), Valid: true},
				Description: sql.NullString{String: dOp.Description, Valid: true},
			})
		}
		if !dberr.IsNotFound(pErr) {
			instances = append(instances, internal.InstanceWithOperation{
				Instance:    v,
				Type:        sql.NullString{String: string(dbmodel.OperationTypeProvision), Valid: true},
				State:       sql.NullString{String: string(pOp.State), Valid: true},
				Description: sql.NullString{String: pOp.Description, Valid: true},
			})
		}
		if !dberr.IsNotFound(uErr) {
			instances = append(instances, internal.InstanceWithOperation{
				Instance:    v,
				Type:        sql.NullString{String: string(dbmodel.OperationTypeUpgradeKyma), Valid: true},
				State:       sql.NullString{String: string(uOp.State), Valid: true},
				Description: sql.NullString{String: uOp.Description, Valid: true},
			})
		}
		if dberr.IsNotFound(dErr) && dberr.IsNotFound(pErr) {
			instances = append(instances, internal.InstanceWithOperation{Instance: v})
		}
	}

	for _, p := range prct {
		p.ApplyToInMemory(instances)
	}

	return instances, nil
}

func (s *Instance) FindAllInstancesForRuntimes(runtimeIdList []string) ([]internal.Instance, error) {
	var instances []internal.Instance

	for _, runtimeID := range runtimeIdList {
		for _, inst := range s.instances {
			if inst.RuntimeID == runtimeID {
				instances = append(instances, inst)
			}
		}
	}

	if len(instances) == 0 {
		return nil, dberr.NotFound("instances with runtime id from list %+q not exist", runtimeIdList)
	}

	return instances, nil
}

func (s *Instance) FindAllInstancesForSubAccounts(subAccountslist []string) ([]internal.Instance, error) {
	var instances []internal.Instance

	for _, subAccount := range subAccountslist {
		for _, inst := range s.instances {
			if inst.SubAccountID == subAccount {
				instances = append(instances, inst)
			}
		}
	}

	return instances, nil
}

func (s *Instance) GetNumberOfInstancesForGlobalAccountID(globalAccountID string) (int, error) {
	numberOfInstances := 0
	for _, inst := range s.instances {
		if inst.GlobalAccountID == globalAccountID {
			numberOfInstances++
		}
	}
	return numberOfInstances, nil
}

func (s *Instance) GetByID(instanceID string) (*internal.Instance, error) {
	inst, ok := s.instances[instanceID]
	if !ok {
		return nil, dberr.NotFound("instance with id %s not exist", instanceID)
	}

	return &inst, nil
}

func (s *Instance) Delete(instanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.instances, instanceID)
	return nil
}

func (s *Instance) Insert(instance internal.Instance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.instances[instance.InstanceID] = instance

	return nil
}

func (s *Instance) Update(instance internal.Instance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.instances[instance.InstanceID] = instance

	return nil
}

func (s *Instance) GetInstanceStats() (internal.InstanceStats, error) {
	return internal.InstanceStats{}, fmt.Errorf("not implemented")
}

func (s *Instance) List(pageSize int, page int) ([]internal.Instance, int, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var toReturn []internal.Instance

	offset := pagination.ConvertPageAndPageSizeToOffset(pageSize, page)

	sortedInstances := getSortedByCreatedAt(s.instances)

	for i := offset; i < offset+pageSize && i < len(sortedInstances); i++ {
		toReturn = append(toReturn, s.instances[sortedInstances[i].InstanceID])
	}

	return toReturn,
		len(toReturn),
		len(s.instances),
		nil
}

func getSortedByCreatedAt(instances map[string]internal.Instance) []internal.Instance {
	instancesArr := make([]internal.Instance, 0, len(instances))
	for _, v := range instances {
		instancesArr = append(instancesArr, v)
	}
	sort.Slice(instancesArr, func(i, j int) bool {
		return instancesArr[i].CreatedAt.Before(instancesArr[j].CreatedAt)
	})
	return instancesArr

}
