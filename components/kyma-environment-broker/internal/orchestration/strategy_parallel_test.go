package orchestration

import (
	"context"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/rand"
)

func TestNewParallelOrchestrationStrategy(t *testing.T) {

	t.Run("immediate schedule", func(t *testing.T) {
		s := NewParallelOrchestrationStrategy(&testExecutor{}, logrus.New())

		startTime, err := time.Parse(maintenanceWindowFormat, "220000+0000")
		require.NoError(t, err)

		n := time.Now()
		start := time.Date(n.Year(), n.Month(), n.Day(), startTime.Hour(), startTime.Minute(), startTime.Second(), startTime.Nanosecond(), &time.Location{})

		ops := make([]internal.RuntimeOperation, 3)
		for i := range ops {
			ops[i] = internal.RuntimeOperation{
				Operation:              internal.Operation{ID: rand.String(5)},
				MaintenanceWindowBegin: start,
			}
		}

		_, err = s.Execute(ops, internal.StrategySpec{Schedule: internal.Immediate})
		assert.NoError(t, err)
	})

	t.Run("maintenance window schedule", func(t *testing.T) {
		s := NewParallelOrchestrationStrategy(&testExecutor{}, logrus.New())

		startTime, err := time.Parse(maintenanceWindowFormat, "030000+0000")
		require.NoError(t, err)

		n := time.Now()
		start := time.Date(n.Year(), n.Month(), n.Day(), startTime.Hour(), startTime.Minute(), startTime.Second(), startTime.Nanosecond(), &time.Location{})

		ops := make([]internal.RuntimeOperation, 3)
		for i := range ops {
			ops[i] = internal.RuntimeOperation{
				Operation:              internal.Operation{ID: rand.String(5)},
				MaintenanceWindowBegin: start,
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		executed := make(chan struct{})

		go func() {
			_, err = s.Execute(ops, internal.StrategySpec{Schedule: internal.MaintenanceWindow})
			assert.NoError(t, err)
			close(executed)
		}()

		select {
		case <-ctx.Done():
		case <-executed:
			t.Fatal("executed method shouldn't finish")
		}
	})
}

type testExecutor struct{}

func (t *testExecutor) Execute(opID string) (time.Duration, error) {
	return 0, nil
}
