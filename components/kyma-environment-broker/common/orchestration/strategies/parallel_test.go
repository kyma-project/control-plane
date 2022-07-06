package strategies

import (
	"sync"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"

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

func (t *testExecutor) Reschedule(operationID string, maintenanceWindowBegin, maintenanceWindowEnd time.Time) error {
	return nil
}

func TestNewParallelOrchestrationStrategy_Immediate(t *testing.T) {
	// given
	executor := &testExecutor{opCalled: map[string]bool{}}
	s := NewParallelOrchestrationStrategy(executor, logrus.New(), 0)

	ops := make([]orchestration.RuntimeOperation, 3)
	for i := range ops {
		ops[i] = orchestration.RuntimeOperation{
			ID: rand.String(5),
		}
	}

	// when
	id, err := s.Execute(ops, orchestration.StrategySpec{Schedule: time.Now().Format(time.RFC3339), Parallel: orchestration.ParallelStrategySpec{Workers: 2}})

	// then
	assert.NoError(t, err)
	s.Wait(id)
}

func TestNewParallelOrchestrationStrategy_MaintenanceWindow(t *testing.T) {
	// given
	executor := &testExecutor{opCalled: map[string]bool{}}
	s := NewParallelOrchestrationStrategy(executor, logrus.New(), 0)

	start := time.Now().Add(3 * time.Second)

	ops := make([]orchestration.RuntimeOperation, 3)
	for i := range ops {
		ops[i] = orchestration.RuntimeOperation{
			ID: rand.String(5),
			Runtime: orchestration.Runtime{
				MaintenanceWindowBegin: start,
				MaintenanceWindowEnd:   start.Add(10 * time.Minute),
			},
		}
	}

	// when
	id, err := s.Execute(ops, orchestration.StrategySpec{Schedule: "imideate", MaintenanceWindow: true, Parallel: orchestration.ParallelStrategySpec{Workers: 2}})

	// then
	assert.NoError(t, err)
	s.Wait(id)
}

func TestNewParallelOrchestrationStrategy_Reschedule(t *testing.T) {
	// given
	executor := &testExecutor{opCalled: map[string]bool{}}
	s := NewParallelOrchestrationStrategy(executor, logrus.New(), 5*time.Second)

	start := time.Now().Add(-5 * time.Second)

	ops := make([]orchestration.RuntimeOperation, 3)
	for i := range ops {
		ops[i] = orchestration.RuntimeOperation{
			ID: rand.String(5),
			Runtime: orchestration.Runtime{
				MaintenanceWindowBegin: start,
				MaintenanceWindowEnd:   start.Add(time.Second),
			},
		}
	}

	// when
	id, err := s.Execute(ops, orchestration.StrategySpec{Schedule: "now", MaintenanceWindow: true, Parallel: orchestration.ParallelStrategySpec{Workers: 2}})

	// then
	assert.NoError(t, err)
	s.Wait(id)
}
