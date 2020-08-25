package kyma_test

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/kyma"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// TODO(upgrade): Cover scenario when resolve returns runtimes or cover it in strategy_*_test.go
func TestUpgradeKymaOrchestration_Execute_Empty(t *testing.T) {
	// given
	store := storage.NewMemoryStorage()

	resolver := &automock.RuntimeResolver{}
	defer resolver.AssertExpectations(t)

	resolver.On("Resolve", internal.TargetSpec{
		Include: nil,
		Exclude: nil,
	}).Return([]internal.Runtime{}, nil)

	id := "id"
	err := store.Orchestration().InsertOrchestration(internal.Orchestration{OrchestrationID: id})
	require.NoError(t, err)

	svc := kyma.NewUpgradeKymaOrchestration(store.Orchestration(), nil, resolver, logrus.New())

	// when
	_, err = svc.Execute(id)
	require.NoError(t, err)
}

func TestUpgradeKymaOrchestration_Execute_InProgress(t *testing.T) {
	// given
	store := storage.NewMemoryStorage()

	resolver := &automock.RuntimeResolver{}
	defer resolver.AssertExpectations(t)

	id := "id"
	err := store.Orchestration().InsertOrchestration(internal.Orchestration{OrchestrationID: id, State: internal.InProgress})
	require.NoError(t, err)

	svc := kyma.NewUpgradeKymaOrchestration(store.Orchestration(), nil, resolver, logrus.New())

	// when
	_, err = svc.Execute(id)
	require.NoError(t, err)
}

func TestUpgradeKymaOrchestration_Execute_InProgressWithRuntimeOperations(t *testing.T) {
	// given
	store := storage.NewMemoryStorage()

	resolver := &automock.RuntimeResolver{}
	defer resolver.AssertExpectations(t)

	id := "id"
	operations := []internal.RuntimeOperation{{
		Runtime: internal.Runtime{
			RuntimeID: id,
		},
		OperationID: id,
	}}
	ops, err := json.Marshal(&operations)
	require.NoError(t, err)

	err = store.Orchestration().InsertOrchestration(
		internal.Orchestration{
			OrchestrationID: id,
			State:           internal.InProgress,
			RuntimeOperations: sql.NullString{
				String: string(ops),
				Valid:  true,
			}})
	require.NoError(t, err)

	svc := kyma.NewUpgradeKymaOrchestration(store.Orchestration(), nil, resolver, logrus.New())

	// when
	_, err = svc.Execute(id)
	require.NoError(t, err)
}
