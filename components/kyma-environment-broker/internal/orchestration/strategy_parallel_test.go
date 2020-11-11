package orchestration

import (
	"sync"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/rand"
)

type testExecutor struct {
	mux      sync.Mutex
	opCalled map[string]bool
}

func (t *testExecutor) Execute(opID string) (time.Duration, error) {
	t.mux.Lock()
	called := t.opCalled[opID]
	t.opCalled[opID] = true
	t.mux.Unlock()

	if !called {
		time.Sleep(1 * time.Second)
		return 1 * time.Second, nil
	} else {
		return 0, nil
	}
}

func TestNewParallelOrchestrationStrategy_Immediate(t *testing.T) {
	// given
	executor := &testExecutor{opCalled: map[string]bool{}}
	s := NewParallelOrchestrationStrategy(executor, logrus.New())

	ops := make([]internal.RuntimeOperation, 3)
	for i := range ops {
		ops[i] = internal.RuntimeOperation{
			Operation: internal.Operation{ID: rand.String(5)},
		}
	}

	// when
	id, err := s.Execute(ops, orchestration.StrategySpec{Schedule: orchestration.Immediate, Parallel: orchestration.ParallelStrategySpec{Workers: 2}})

	// then
	assert.NoError(t, err)
	s.Wait(id)
}

func TestNewParallelOrchestrationStrategy_MaintenanceWindow(t *testing.T) {
	// given
	executor := &testExecutor{opCalled: map[string]bool{}}
	s := NewParallelOrchestrationStrategy(executor, logrus.New())

	start := time.Now().Add(5 * time.Second)

	ops := make([]internal.RuntimeOperation, 3)
	for i := range ops {
		ops[i] = internal.RuntimeOperation{
			Operation:              internal.Operation{ID: rand.String(5)},
			MaintenanceWindowBegin: start,
		}
	}

	// when
	id, err := s.Execute(ops, orchestration.StrategySpec{Schedule: orchestration.MaintenanceWindow, Parallel: orchestration.ParallelStrategySpec{Workers: 2}})

	// then
	assert.NoError(t, err)
	s.Wait(id)
}
