package orchestration

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

//go:generate mockery -name=RuntimeResolver -output=automock -outpkg=automock -case=underscore
// RuntimeResolver given an input slice of target specs to include and exclude, resolves and returns a list of unique Runtime objects.
type RuntimeResolver interface {
	Resolve(targets orchestration.TargetSpec) ([]internal.Runtime, error)
}

//go:generate mockery -name=Strategy -output=automock -outpkg=automock -case=underscore
// Strategy interface encapsulates the strategy how the orchestration is performed.
type Strategy interface {
	// Execute invokes operation managers' Execute(operationID string) method for each operation according to the encapsulated strategy.
	// The strategy is executed asynchronously. Successful call to the function returns a unique identifier, which can be used in a subsequent call to Wait().
	Execute(operations []internal.RuntimeOperation, strategySpec orchestration.StrategySpec) (string, error)
	// Wait blocks and waits until the execution with the given ID is finished.
	Wait(executionID string)
}
