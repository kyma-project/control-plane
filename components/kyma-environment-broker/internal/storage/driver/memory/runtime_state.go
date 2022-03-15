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

func (s *runtimeState) GetLatestByRuntimeID(runtimeID string) (internal.RuntimeState, error) {
	states, err := s.getRuntimeStatesByRuntimeID(runtimeID)
	if err != nil {
		return internal.RuntimeState{}, err
	}

	return states[0], nil
}

func (s *runtimeState) GetLatestWithKymaVersionByRuntimeID(runtimeID string) (internal.RuntimeState, error) {
	states, err := s.getRuntimeStatesByRuntimeID(runtimeID)
	if err != nil {
		return internal.RuntimeState{}, err
	}

	for _, state := range states {
		if state.ClusterSetup != nil && state.ClusterSetup.KymaConfig.Version != "" {
			return state, nil
		}
		if state.KymaConfig.Version != "" {
			return state, nil
		}
	}

	return internal.RuntimeState{}, dberr.NotFound("runtime state with Reconciler input for runtime with ID: %s not found", runtimeID)
}

func (s *runtimeState) GetLatestWithReconcilerInputByRuntimeID(runtimeID string) (internal.RuntimeState, error) {
	states, err := s.getRuntimeStatesByRuntimeID(runtimeID)
	if err != nil {
		return internal.RuntimeState{}, err
	}

	for _, state := range states {
		if state.ClusterSetup != nil {
			return state, nil
		}
	}

	return internal.RuntimeState{}, dberr.NotFound("runtime state with Reconciler input for runtime with ID: %s not found", runtimeID)
}

func (s *runtimeState) GetLatestWithOIDCConfigByRuntimeID(runtimeID string) (internal.RuntimeState, error) {
	states, err := s.getRuntimeStatesByRuntimeID(runtimeID)
	if err != nil {
		return internal.RuntimeState{}, err
	}

	for _, state := range states {
		if state.ClusterConfig.OidcConfig != nil {
			return state, nil
		}
	}

	return internal.RuntimeState{}, dberr.NotFound("runtime state with OIDC config for runtime with ID: %s not found", runtimeID)
}

func (s *runtimeState) getRuntimeStatesByRuntimeID(runtimeID string) ([]internal.RuntimeState, error) {
	states, err := s.ListByRuntimeID(runtimeID)
	if err != nil {
		return nil, err
	}
	if len(states) == 0 {
		return nil, dberr.NotFound("runtime state for runtime with ID: %s not found", runtimeID)
	}
	return states, nil
}
