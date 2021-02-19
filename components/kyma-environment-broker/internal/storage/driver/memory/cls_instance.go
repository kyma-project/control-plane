package memory

import (
	"sync"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
)

type clsInstances struct {
	mu sync.Mutex

	data map[string]internal.CLSInstance
}

type clsKey struct {
	GlobalAccountID string
}

func NewCLSInstances() *clsInstances {
	return &clsInstances{
		mu:   sync.Mutex{},
		data: make(map[string]internal.CLSInstance),
	}
}

func (s *clsInstances) FindActiveByGlobalAccountID(globalAccountID string) (*internal.CLSInstance, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	exists := false
	var instance internal.CLSInstance
	for _, v := range s.data {
		if v.GlobalAccountID == globalAccountID {
			exists = true
			instance = v
		}
	}

	if !exists {
		return nil, false, nil
	}

	return &instance, true, nil
}

func (s *clsInstances) FindByID(clsInstanceID string) (*internal.CLSInstance, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, exists := s.data[clsInstanceID]
	if !exists {
		return nil, false, nil
	}

	return &instance, true, nil
}

func (s *clsInstances) Insert(instance internal.CLSInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	clsInstanceID := instance.ID
	if _, exists := s.data[clsInstanceID]; exists {
		return dberr.AlreadyExists("instance already exists")
	}
	s.data[clsInstanceID] = instance

	return nil
}

func (s *clsInstances) Reference(version int, clsInstanceID, skrInstanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, exists := s.data[clsInstanceID]
	if !exists {
		return dberr.NotFound("instance not found")
	}

	instance.ReferencedSKRInstanceIDs = append(instance.ReferencedSKRInstanceIDs, skrInstanceID)

	return nil
}

func (s *clsInstances) Unreference(version int, clsInstanceID, skrInstanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, exists := s.data[clsInstanceID]
	if !exists {
		return dberr.NotFound("instance not found")
	}

	if !exists {
		return dberr.NotFound("instance not found")
	}

	refsCopy := make([]string, len(instance.ReferencedSKRInstanceIDs))
	copy(refsCopy, instance.ReferencedSKRInstanceIDs)
	instance.ReferencedSKRInstanceIDs = nil

	for _, ref := range refsCopy {
		if ref != skrInstanceID {
			instance.ReferencedSKRInstanceIDs = append(instance.ReferencedSKRInstanceIDs, ref)
		}
	}

	return nil
}

func (s *clsInstances) MarkAsBeingRemoved(version int, clsInstanceID, skrInstanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, exists := s.data[clsInstanceID]
	if !exists {
		return dberr.NotFound("instance not found")
	}

	if len(instance.RemovedBySKRInstanceID) > 0 && instance.RemovedBySKRInstanceID != skrInstanceID {
		return dberr.Conflict("remover skr instance id is already set to another value")
	}

	instance.RemovedBySKRInstanceID = skrInstanceID

	return nil
}

func (s *clsInstances) Remove(clsInstanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.data[clsInstanceID]
	if !exists {
		return dberr.NotFound("instance not found")
	}

	delete(s.data, clsInstanceID)

	return nil
}
