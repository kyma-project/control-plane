package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFinishStage(t *testing.T) {
	operation, error := NewProvisioningOperation("1", ProvisioningParameters{})
	assert.NoError(t, error)
	assert.NotEmpty(t, operation)

	t.Run("should not add empty stage", func(t *testing.T) {
		stage := ""
		operation.FinishStage(stage)
		foundStages := countStageOccurrences(operation, stage)
		assert.Equal(t, 0, foundStages)
	})

	t.Run("should add one unique stage", func(t *testing.T) {
		stage := "start"
		operation.FinishStage(stage)
		foundStages := countStageOccurrences(operation, stage)
		assert.Equal(t, 1, foundStages)
	})

	t.Run("should not add duplicated stages", func(t *testing.T) {
		stage := "create_runtime"
		operation.FinishStage(stage)
		operation.FinishStage(stage)
		foundStages := countStageOccurrences(operation, stage)
		assert.Equal(t, 1, foundStages)
	})

	t.Run("should add two distinct stages", func(t *testing.T) {
		stage1 := "start"
		stage2 := "create_runtime"
		operation.FinishStage(stage1)
		operation.FinishStage(stage2)
		foundStages1 := countStageOccurrences(operation, stage1)
		foundStages2 := countStageOccurrences(operation, stage2)
		assert.Equal(t, 1, foundStages1)
		assert.Equal(t, 1, foundStages2)
	})
}

func countStageOccurrences(operation ProvisioningOperation, stage string) int {
	foundStages := 0
	for _, v := range operation.FinishedStages {
		if v == stage {
			foundStages++
		}
	}
	return foundStages
}
