package memory

import (
	"sync"

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
	result := make([]internal.RuntimeState, 0)

	for _, state := range s.runtimeStates {
		if state.RuntimeID == runtimeID {
			result = append(result, state)
		}
	}

	return result, nil
}
