package handlers_test

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConverter_OrchestrationToDTO(t *testing.T) {
	// given
	c := handlers.Converter{}

	id := "id"
	givenOrchestration := &internal.Orchestration{OrchestrationID: id}

	// when
	resp, err := c.OrchestrationToDTO(givenOrchestration)

	// then
	require.NoError(t, err)
	assert.Equal(t, id, resp.OrchestrationID)
}

func TestConverter_OrchestrationListToDTO(t *testing.T) {
	// given
	c := handlers.Converter{}

	id := "id"
	givenOrchestration := []internal.Orchestration{{OrchestrationID: id}}

	// when
	resp, err := c.OrchestrationListToDTO(givenOrchestration)

	// then
	require.NoError(t, err)
	assert.Equal(t, len(givenOrchestration), len(resp.Orchestrations))
}
