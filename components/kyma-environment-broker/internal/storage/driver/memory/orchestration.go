package memory

import (
	"sort"
	"sync"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/pagination"

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
	s.mu.Lock()
	defer s.mu.Unlock()

	inst, ok := s.orchestrations[orchestrationID]
	if !ok {
		return nil, dberr.NotFound("orchestration with id %s not exist", orchestrationID)
	}

	return &inst, nil
}

func (s *orchestration) List(pageSize int, page int) ([]internal.Orchestration, int, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]internal.Orchestration, 0)
	offset := pagination.ConvertPageAndPageSizeToOffset(pageSize, page)

	sortedOrchestrations := s.getSortedByCreatedAt(s.orchestrations)

	for i := offset; i < offset+pageSize && i < len(sortedOrchestrations); i++ {
		result = append(result, s.orchestrations[sortedOrchestrations[i].OrchestrationID])
	}

	return result,
		len(result),
		len(s.orchestrations),
		nil
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

func (s *orchestration) getSortedByCreatedAt(orchestrations map[string]internal.Orchestration) []internal.Orchestration {
	orchestrationsList := make([]internal.Orchestration, 0, len(orchestrations))
	for _, v := range orchestrations {
		orchestrationsList = append(orchestrationsList, v)
	}
	sort.Slice(orchestrationsList, func(i, j int) bool {
		return orchestrationsList[i].CreatedAt.Before(orchestrationsList[j].CreatedAt)
	})
	return orchestrationsList
}
