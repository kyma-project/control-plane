package orchestration

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/rand"
)

func TestNewInstantOrchestrationStrategy(t *testing.T) {
	s := NewInstantOrchestrationStrategy(&testExecutor{}, logrus.New())

	ops := make([]internal.RuntimeOperation, 3)
	for i := range ops {
		ops[i] = internal.RuntimeOperation{OperationID: rand.String(3)}
	}

	_, err := s.Execute(ops, internal.StrategySpec{})
	assert.NoError(t, err)
}

type testExecutor struct{}

func (t *testExecutor) Execute(opID string) (time.Duration, error) {
	return 0, nil
}
