package memory

import (
	"sort"
	"sync"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/pagination"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession/dbmodel"
)

type orchestrations struct {
	mu sync.Mutex

	orchestrations map[string]orchestration.Orchestration
}

func NewOrchestrations() *orchestrations {
	return &orchestrations{
		orchestrations: make(map[string]orchestration.Orchestration, 0),
	}
}

func (s *orchestrations) Insert(orchestration orchestration.Orchestration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orchestrations[orchestration.OrchestrationID] = orchestration

	return nil
}

func (s *orchestrations) GetByID(orchestrationID string) (*orchestration.Orchestration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inst, ok := s.orchestrations[orchestrationID]
	if !ok {
		return nil, dberr.NotFound("orchestration with id %s not exist", orchestrationID)
	}

	return &inst, nil
}

func (s *orchestrations) List(filter dbmodel.OrchestrationFilter) ([]orchestration.Orchestration, int, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]orchestration.Orchestration, 0)
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

func (s *orchestrations) Update(orchestration orchestration.Orchestration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.orchestrations[orchestration.OrchestrationID]; !ok {
		return dberr.NotFound("orchestration with id %s not exist", orchestration.OrchestrationID)

	}
	s.orchestrations[orchestration.OrchestrationID] = orchestration

	return nil
}

func (s *orchestrations) ListByState(state string) ([]orchestration.Orchestration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]orchestration.Orchestration, 0)

	for _, o := range s.orchestrations {
		if o.State == state {
			result = append(result, o)
		}
	}

	return result, nil
}

func (s *orchestrations) sortByCreatedAt(orchestrations []orchestration.Orchestration) {
	sort.Slice(orchestrations, func(i, j int) bool {
		return orchestrations[i].CreatedAt.Before(orchestrations[j].CreatedAt)
	})
}

func (s *orchestrations) filter(filter dbmodel.OrchestrationFilter) []orchestration.Orchestration {
	orchestrations := make([]orchestration.Orchestration, 0, len(s.orchestrations))
	equal := func(a, b string) bool { return a == b }
	for _, v := range s.orchestrations {
		if ok := matchFilter(v.State, filter.States, equal); !ok {
			continue
		}

		orchestrations = append(orchestrations, v)
	}

	return orchestrations
}
