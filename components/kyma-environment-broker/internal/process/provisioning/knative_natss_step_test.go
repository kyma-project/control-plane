package provisioning

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestKnativeNatss(t *testing.T) {
	// Given
	memoryStorage := storage.NewMemoryStorage()
	log := logrus.New()
	operation := fixOperationWithPlanID(t, "any")
	simpleInputCreator := newInputCreator()
	operation.InputCreator = simpleInputCreator
	var runTime time.Duration = 0

	step := NewKnativeProvisionerNatssStep(memoryStorage.Operations())

	// When
	returnedOperation, time, err := step.Run(operation, log)

	// Then
	require.NoError(t, err)
	simpleInputCreator.AssertEnabledComponent(t, KebComponentNameKnativeProvisionerNatss)
	assert.Equal(t, runTime, time)
	assert.Equal(t, operation, returnedOperation)
}
