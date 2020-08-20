package provisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var (
	cei = gqlschema.ConfigEntryInput{
		Key:    "global.natsStreaming.persistence.enabled",
		Value:  "true",
		Secret: nil,
	}
	ceo = gqlschema.ConfigEntryInput{
		Key:    "global.natsStreaming.persistence.enabled",
		Value:  "false",
		Secret: nil,
	}
)

func TestNatssWithInitialOverrides(t *testing.T) {
	// Given
	memoryStorage := storage.NewMemoryStorage()
	log := logrus.New()
	operation := fixOperationWithPlanID(t, "any")
	simpleInputCreator := newInputCreator()
	simpleInputCreator.AppendOverrides(components.NatsStreaming, []*gqlschema.ConfigEntryInput{&cei})
	operation.InputCreator = simpleInputCreator
	var runTime time.Duration = 0

	step := NewNatsStreamingOverridesStep(memoryStorage.Operations())

	// When
	returnedOperation, time, err := step.Run(operation, log)

	// Then
	require.NoError(t, err)
	ovrs := simpleInputCreator.overrides[components.NatsStreaming]
	assert.Equal(t, &cei, ovrs[0])
	assert.Equal(t, &ceo, ovrs[1])
	assert.Equal(t, runTime, time)
	assert.Equal(t, operation, returnedOperation)
}

func TestNatssWithEmptyOverrides(t *testing.T) {
	// Given
	memoryStorage := storage.NewMemoryStorage()
	log := logrus.New()
	operation := fixOperationWithPlanID(t, "any")
	simpleInputCreator := newInputCreator()
	simpleInputCreator.AssertNoOverrides(t)
	operation.InputCreator = simpleInputCreator
	var runTime time.Duration = 0

	step := NewNatsStreamingOverridesStep(memoryStorage.Operations())

	// When
	returnedOperation, time, err := step.Run(operation, log)

	// Then
	require.NoError(t, err)
	simpleInputCreator.AssertOverride(t, components.NatsStreaming, ceo)
	assert.Equal(t, runTime, time)
	assert.Equal(t, operation, returnedOperation)
}
