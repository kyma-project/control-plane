package handlers_test

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

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
	stats := map[string]int{"in progress": 5, "succeeded": 3}

	// when
	resp, err := c.OrchestrationToDTO(givenOrchestration, stats)

	// then
	require.NoError(t, err)
	assert.Equal(t, id, resp.OrchestrationID)
	assert.Equal(t, stats, resp.OperationStats)
}

func TestConverter_OrchestrationListToDTO(t *testing.T) {
	// given
	c := handlers.Converter{}

	id := "id"
	givenOrchestration := []internal.Orchestration{{OrchestrationID: id}}

	// when
	resp, err := c.OrchestrationListToDTO(givenOrchestration, 1, 5)

	// then
	require.NoError(t, err)
	assert.Equal(t, len(givenOrchestration), len(resp.Data))
	assert.Equal(t, id, resp.Data[0].OrchestrationID)
	assert.Equal(t, 1, resp.Count)
	assert.Equal(t, 5, resp.TotalCount)
}

func TestConverter_UpgradeKymaOperationToDTO(t *testing.T) {
	// given
	c := handlers.Converter{}

	id := "id"
	givenOperation := fixOperation(id)

	// when
	resp, err := c.UpgradeKymaOperationToDTO(givenOperation)

	// then
	require.NoError(t, err)
	assert.Equal(t, id, resp.OrchestrationID)
}

func TestConverter_UpgradeKymaOperationListToDTO(t *testing.T) {
	// given
	c := handlers.Converter{}

	id := "id"
	givenOperations := []internal.UpgradeKymaOperation{
		fixOperation(id),
		fixOperation("another"),
	}

	// when
	resp, err := c.UpgradeKymaOperationListToDTO(givenOperations, 2, 5)

	// then
	require.NoError(t, err)
	require.Len(t, resp.Data, 2)
	assert.Equal(t, id, resp.Data[0].OrchestrationID)
	assert.Equal(t, 2, resp.Count)
	assert.Equal(t, 5, resp.TotalCount)
}

func TestConverter_UpgradeKymaOperationToDetailDTO(t *testing.T) {
	// given
	c := handlers.Converter{}

	id := "id"
	givenOperation := fixOperation(id)
	kymaConfig := gqlschema.KymaConfigInput{Version: id}
	clusterConfig := gqlschema.GardenerConfigInput{KubernetesVersion: id}

	// when
	resp, err := c.UpgradeKymaOperationToDetailDTO(givenOperation, kymaConfig, clusterConfig)

	// then
	require.NoError(t, err)
	assert.Equal(t, id, resp.OrchestrationID)
	assert.Equal(t, id, resp.KymaConfig.Version)
	assert.Equal(t, id, resp.ClusterConfig.KubernetesVersion)
}

func fixOperation(id string) internal.UpgradeKymaOperation {
	upgradeOperation := fixture.FixUpgradeKymaOperation("", "")
	upgradeOperation.OrchestrationID = id

	return upgradeOperation
}
