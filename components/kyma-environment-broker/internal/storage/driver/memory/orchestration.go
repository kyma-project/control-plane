package memory

import (
	"sort"
	"sync"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/pagination"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession/dbmodel"
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

func (s *orchestration) List(filter dbmodel.OrchestrationFilter) ([]internal.Orchestration, int, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]internal.Orchestration, 0)
	offset := pagination.ConvertPageAndPageSizeToOffset(filter.PageSize, filter.Page)

	orchestrations := s.filter(filter)
	s.sortByCreatedAt(orchestrations)

	for i := offset; (filter.PageSize < 1 || i < offset+filter.PageSize) && i < len(orchestrations); i++ {
		result = append(result, s.orchestrations[orchestrations[i].OrchestrationID])
	}

	return result,
		len(result),
		len(orchestrations),
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

func (s *orchestration) sortByCreatedAt(orchestrations []internal.Orchestration) {
	sort.Slice(orchestrations, func(i, j int) bool {
		return orchestrations[i].CreatedAt.Before(orchestrations[j].CreatedAt)
	})
}

func (s *orchestration) filter(filter dbmodel.OrchestrationFilter) []internal.Orchestration {
	orchestrations := make([]internal.Orchestration, 0, len(s.orchestrations))
	equal := func(a, b string) bool { return a == b }
	for _, v := range s.orchestrations {
		if ok := matchFilter(v.State, filter.States, equal); !ok {
			continue
		}

		orchestrations = append(orchestrations, v)
	}

	return orchestrations
}
