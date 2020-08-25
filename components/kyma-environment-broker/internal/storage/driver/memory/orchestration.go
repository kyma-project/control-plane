package memory

import (
	"sync"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
)

type orchestration struct {
	mu sync.Mutex

	orchestrations map[string]internal.Orchestration
}

func NewOrchestration() *orchestration {
	return &orchestration{
		orchestrations: make(map[string]internal.Orchestration, 0),
	}
}

func (s *orchestration) InsertOrchestration(orchestration internal.Orchestration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orchestrations[orchestration.OrchestrationID] = orchestration

	return nil
}
func (s *orchestration) GetOrchestrationByID(orchestrationID string) (*internal.Orchestration, error) {
	inst, ok := s.orchestrations[orchestrationID]
	if !ok {
		return nil, dberr.NotFound("orchestration with id %s not exist", orchestrationID)
	}

	return &inst, nil
}
func (s *orchestration) UpdateOrchestration(orchestration internal.Orchestration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.orchestrations[orchestration.OrchestrationID]; !ok {
		return dberr.NotFound("orchestration with id %s not exist", orchestration.OrchestrationID)

	}

	s.orchestrations[orchestration.OrchestrationID] = orchestration

	return nil
}
