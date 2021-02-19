package memory

import (
	"database/sql"
	"errors"
	"regexp"
	"sort"
	"sync"

	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/pagination"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/predicate"
)

type instances struct {
	mu                sync.Mutex
	instances         map[string]internal.Instance
	operationsStorage *operations
}

func NewInstance(operations *operations) *instances {
	return &instances{
		instances:         make(map[string]internal.Instance, 0),
		operationsStorage: operations,
	}
}

func (s *instances) InsertWithoutEncryption(instance internal.Instance) error {
	return errors.New("not implemented")
}
func (s *instances) UpdateWithoutEncryption(instance internal.Instance) (*internal.Instance, error) {
	return nil, errors.New("not implemented")
}
func (s *instances) ListWithoutDecryption(dbmodel.InstanceFilter) ([]internal.Instance, int, int, error) {
	return nil, 0, 0, errors.New("not implemented")
}

func (s *instances) FindAllJoinedWithOperations(prct ...predicate.Predicate) ([]internal.InstanceWithOperation, error) {
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

func (s *instances) FindAllInstancesForRuntimes(runtimeIdList []string) ([]internal.Instance, error) {
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

func (s *instances) FindAllInstancesForSubAccounts(subAccountslist []string) ([]internal.Instance, error) {
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

func (s *instances) GetNumberOfInstancesForGlobalAccountID(globalAccountID string) (int, error) {
	numberOfInstances := 0
	for _, inst := range s.instances {
		if inst.GlobalAccountID == globalAccountID {
			numberOfInstances++
		}
	}
	return numberOfInstances, nil
}

func (s *instances) GetByID(instanceID string) (*internal.Instance, error) {
	inst, ok := s.instances[instanceID]
	if !ok {
		return nil, dberr.NotFound("instance with id %s not exist", instanceID)
	}

	return &inst, nil
}

func (s *instances) Delete(instanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.instances, instanceID)
	return nil
}

func (s *instances) Insert(instance internal.Instance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.instances[instance.InstanceID] = instance

	return nil
}

func (s *instances) Update(instance internal.Instance) (*internal.Instance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	oldInst, exists := s.instances[instance.InstanceID]
	if !exists {
		return nil, dberr.NotFound("instance %s not found", instance.InstanceID)
	}
	if oldInst.Version != instance.Version {
		return nil, dberr.Conflict("unable to update instance %s - conflict", instance.InstanceID)
	}
	instance.Version = instance.Version + 1
	s.instances[instance.InstanceID] = instance

	return &instance, nil
}

func (s *instances) GetInstanceStats() (internal.InstanceStats, error) {
	return internal.InstanceStats{}, fmt.Errorf("not implemented")
}

func (s *instances) List(filter dbmodel.InstanceFilter) ([]internal.Instance, int, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var toReturn []internal.Instance

	offset := pagination.ConvertPageAndPageSizeToOffset(filter.PageSize, filter.Page)

	instances := s.filterInstances(filter)
	sortInstancesByCreatedAt(instances)

	for i := offset; (filter.PageSize < 1 || i < offset+filter.PageSize) && i < len(instances); i++ {
		toReturn = append(toReturn, s.instances[instances[i].InstanceID])
	}

	return toReturn,
		len(toReturn),
		len(instances),
		nil
}

func sortInstancesByCreatedAt(instances []internal.Instance) {
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].CreatedAt.Before(instances[j].CreatedAt)
	})
}

func (s *instances) filterInstances(filter dbmodel.InstanceFilter) []internal.Instance {
	inst := make([]internal.Instance, 0, len(s.instances))
	var ok bool
	equal := func(a, b string) bool {
		return a == b
	}
	domainMatch := func(url, filter string) bool {
		// Preceeding character is either a . or / (after protocol://)
		// match subdomain inputs
		// match any .upperdomain zero or more times
		matchExpr := fmt.Sprintf(`[./]%s(\.[0-9A-Za-z-]+)*$`, filter)
		matched, err := regexp.MatchString(matchExpr, url)
		return err == nil && matched
	}

	for _, v := range s.instances {
		if ok = matchFilter(v.InstanceID, filter.InstanceIDs, equal); !ok {
			continue
		}
		if ok = matchFilter(v.GlobalAccountID, filter.GlobalAccountIDs, equal); !ok {
			continue
		}
		if ok = matchFilter(v.SubAccountID, filter.SubAccountIDs, equal); !ok {
			continue
		}
		if ok = matchFilter(v.RuntimeID, filter.RuntimeIDs, equal); !ok {
			continue
		}
		if ok = matchFilter(v.ServicePlanName, filter.Plans, equal); !ok {
			continue
		}
		if ok = matchFilter(v.ProviderRegion, filter.Regions, equal); !ok {
			continue
		}
		// Match domains with dashboard url
		if ok = matchFilter(v.DashboardURL, filter.Domains, domainMatch); !ok {
			continue
		}

		inst = append(inst, v)
	}

	return inst
}

func matchFilter(value string, filters []string, match func(string, string) bool) bool {
	if len(filters) == 0 {
		return true
	}
	for _, f := range filters {
		if match(value, f) {
			return true
		}
	}
	return false
}
