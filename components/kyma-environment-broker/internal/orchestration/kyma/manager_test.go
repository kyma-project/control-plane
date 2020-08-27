package kyma_test

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/kyma"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

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
	err := store.Orchestrations().Insert(internal.Orchestration{OrchestrationID: id})
	require.NoError(t, err)

	svc := kyma.NewUpgradeKymaManager(store.Orchestrations(), nil, resolver, logrus.New())

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
	err := store.Orchestrations().Insert(internal.Orchestration{OrchestrationID: id, State: internal.InProgress})
	require.NoError(t, err)

	svc := kyma.NewUpgradeKymaManager(store.Orchestrations(), nil, resolver, logrus.New())

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

	givenO := internal.Orchestration{
		OrchestrationID: id,
		State:           internal.InProgress,
		RuntimeOperations: sql.NullString{
			String: string(ops),
			Valid:  true,
		}}
	err = store.Orchestrations().Insert(givenO)
	require.NoError(t, err)

	svc := kyma.NewUpgradeKymaManager(store.Orchestrations(), &testExecutor{}, resolver, logrus.New())

	// when
	_, err = svc.Execute(id)
	require.NoError(t, err)

	o, err := store.Orchestrations().GetByID(id)
	require.NoError(t, err)

	assert.Equal(t, internal.Succeeded, o.State)
}

type testExecutor struct{}

func (t *testExecutor) Execute(opID string) (time.Duration, error) {
	return 0, nil
}
