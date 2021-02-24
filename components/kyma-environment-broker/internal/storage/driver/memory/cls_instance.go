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

func (s *clsInstances) Reference(version int, globalAccountID, skrInstanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := clsKey{GlobalAccountID: globalAccountID}
	instance, exists := s.data[k]
	if !exists {
		return dberr.NotFound("instance not found")
	}

	instance.ReferencedSKRInstanceIDs = append(instance.ReferencedSKRInstanceIDs, skrInstanceID)

	return nil
}

func (s *clsInstances) Unreference(version int, globalAccountID, skrInstanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := clsKey{GlobalAccountID: globalAccountID}
	instance, exists := s.data[k]
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

func (s *clsInstances) MarkAsBeingRemoved(version int, globalAccountID, skrInstanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := clsKey{GlobalAccountID: globalAccountID}
	instance, exists := s.data[k]
	if !exists {
		return dberr.NotFound("instance not found")
	}

	if len(instance.RemovedBySKRInstanceID) > 0 && instance.RemovedBySKRInstanceID != skrInstanceID {
		return dberr.Conflict("remover skr instance id is already set to another value")
	}

	instance.RemovedBySKRInstanceID = skrInstanceID

	return nil
}

func (s *clsInstances) RemoveInstance(version int, globalAccountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := clsKey{GlobalAccountID: globalAccountID}

	_, exists := s.data[k]
	if !exists {
		return dberr.NotFound("instance not found")
	}

	delete(s.data, k)

	return nil
}
