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
	Name string
}

func NewCLSInstances() *clsInstances {
	return &clsInstances{
		data: make(map[clsKey]internal.CLSInstance, 0),
	}
}

func (s *clsInstances) FindInstance(name string) (internal.CLSInstance, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := clsKey{Name: name}
	instance, exists := s.data[k]
	if !exists {
		return internal.CLSInstance{}, false, nil
	}

	return instance, true, nil
}

func (s *clsInstances) InsertInstance(instance internal.CLSInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := clsKey{Name: instance.Name}
	if _, exists := s.data[k]; exists {
		return dberr.AlreadyExists("instance already exists")
	}
	s.data[k] = instance

	return nil
}
