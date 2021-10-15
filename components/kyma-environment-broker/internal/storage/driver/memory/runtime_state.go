package memory

import (
	"sort"
	"sync"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type runtimeState struct {
	mu sync.Mutex

	runtimeStates map[string]internal.RuntimeState
}

func NewRuntimeStates() *runtimeState {
	return &runtimeState{
		runtimeStates: make(map[string]internal.RuntimeState, 0),
	}
}

func (s *runtimeState) Insert(runtimeState internal.RuntimeState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runtimeStates[runtimeState.ID] = runtimeState

	return nil
}

func (s *runtimeState) ListByRuntimeID(runtimeID string) ([]internal.RuntimeState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]internal.RuntimeState, 0)

	for _, state := range s.runtimeStates {
		if state.RuntimeID == runtimeID {
			result = append(result, state)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (s *runtimeState) GetByOperationID(operationID string) (internal.RuntimeState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, rs := range s.runtimeStates {
		if rs.OperationID == operationID {
			return rs, nil
		}
	}

	return internal.RuntimeState{}, dberr.NotFound("runtime state with operation ID %s not found", operationID)
}

func (s *runtimeState) GetLastByRuntimeID(runtimeID string) (internal.RuntimeState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	states, err := s.ListByRuntimeID(runtimeID)
	if err != nil {
		return internal.RuntimeState{}, err
	}

	return states[0], nil
}
