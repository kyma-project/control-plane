package memory

import (
	"database/sql"
	"sort"
	"sync"

	"github.com/kyma-incubator/compass/components/director/pkg/pagination"

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

func (s *Instance) List(limit int, cursor string) ([]internal.Instance, *pagination.Page, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var toReturn []internal.Instance

	offset, err := pagination.DecodeOffsetCursor(cursor)
	if err != nil {
		return nil, nil, 0, err
	}
	keys := getSortedKeys(s.instances)

	for i := offset; i < offset+limit; i++ {
		toReturn = append(toReturn, s.instances[keys[offset]])
	}

	hasNextPage := false
	endCursor := ""
	if len(s.instances) > offset+len(toReturn) {
		hasNextPage = true
		endCursor = pagination.EncodeNextOffsetCursor(offset, limit)
	}

	return toReturn,
		&pagination.Page{
			StartCursor: cursor,
			EndCursor:   endCursor,
			HasNextPage: hasNextPage,
		},
		len(s.instances),
		nil
}

func getSortedKeys(instances map[string]internal.Instance) []string {
	keys := make([]string, 0, len(instances))
	for k := range instances {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys

}
