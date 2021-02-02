package memory

import (
	"sync"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
)

type clsInstances struct {
	mu sync.Mutex

	data map[clsKey]internal.CLSInstance
}

type clsKey struct {
	Name   string
	Region string
}

func NewCLSInstances() *clsInstances {
	return &clsInstances{
		data: make(map[clsKey]internal.CLSInstance, 0),
	}
}

func (s *clsInstances) FindInstanceByName(name, region string) (internal.CLSInstance, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := clsKey{Name: name, Region: region}
	tenant, exists := s.data[k]
	if !exists {
		return internal.CLSInstance{}, false, nil
	}

	return tenant, true, nil
}

func (s *clsInstances) InsertInstance(tenant internal.CLSInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := clsKey{Name: tenant.Name, Region: tenant.Region}
	if _, exists := s.data[k]; exists {
		return dberr.AlreadyExists("tenant already exists")
	}
	s.data[k] = tenant

	return nil
}
