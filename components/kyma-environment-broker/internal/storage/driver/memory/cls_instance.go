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
	GlobalAccountID string
}

func NewCLSInstances() *clsInstances {
	return &clsInstances{
		data: make(map[clsKey]internal.CLSInstance, 0),
	}
}

func (s *clsInstances) FindInstance(globalAccountID string) (*internal.CLSInstance, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := clsKey{GlobalAccountID: globalAccountID}
	instance, exists := s.data[k]
	if !exists {
		return nil, false, nil
	}

	return &instance, true, nil
}

func (s *clsInstances) InsertInstance(instance internal.CLSInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := clsKey{GlobalAccountID: instance.GlobalAccountID}
	if _, exists := s.data[k]; exists {
		return dberr.AlreadyExists("instance already exists")
	}
	s.data[k] = instance

	return nil
}

func (s *clsInstances) AddReference(globalAccountID, skrInstanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := clsKey{GlobalAccountID: globalAccountID}
	instance, exists := s.data[k]
	if !exists {
		return dberr.NotFound("instance not found")
	}

	instance.SKRReferences = append(instance.SKRReferences, skrInstanceID)

	return nil
}
