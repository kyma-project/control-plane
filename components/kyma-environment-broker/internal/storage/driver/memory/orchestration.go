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

func NewOrchestrations() *orchestration {
	return &orchestration{
		orchestrations: make(map[string]internal.Orchestration, 0),
	}
}

func (s *orchestration) Insert(orchestration internal.Orchestration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orchestrations[orchestration.OrchestrationID] = orchestration

	return nil
}

func (s *orchestration) GetByID(orchestrationID string) (*internal.Orchestration, error) {
	inst, ok := s.orchestrations[orchestrationID]
	if !ok {
		return nil, dberr.NotFound("orchestration with id %s not exist", orchestrationID)
	}

	return &inst, nil
}

func (s *orchestration) ListAll() ([]internal.Orchestration, error) {
	result := make([]internal.Orchestration, 0)
	for _, o := range s.orchestrations {
		result = append(result, o)
	}

	return result, nil
}

func (s *orchestration) Update(orchestration internal.Orchestration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.orchestrations[orchestration.OrchestrationID]; !ok {
		return dberr.NotFound("orchestration with id %s not exist", orchestration.OrchestrationID)

	}
	s.orchestrations[orchestration.OrchestrationID] = orchestration

	return nil
}

func (s *orchestration) ListByState(state string) ([]internal.Orchestration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]internal.Orchestration, 0)

	for _, o := range s.orchestrations {
		if o.State == state {
			result = append(result, o)
		}
	}

	return result, nil
}
