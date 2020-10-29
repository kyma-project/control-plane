package orchestration

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/rand"
)

func TestNewParallelOrchestrationStrategy(t *testing.T) {

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

	_, err = s.Execute(ops, orchestration.StrategySpec{Schedule: orchestration.Immediate})
	assert.NoError(t, err)

	_, err = s.Execute(ops, orchestration.StrategySpec{Schedule: orchestration.MaintenanceWindow})
	assert.NoError(t, err)
}

type testExecutor struct{}

func (t *testExecutor) Execute(opID string) (time.Duration, error) {
	return 0, nil
}
