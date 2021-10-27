package postsql_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeState(t *testing.T) {

	ctx := context.Background()

	t.Run("should insert and fetch RuntimeState", func(t *testing.T) {
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		tablesCleanupFunc, err := storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)
		defer tablesCleanupFunc()

		cipher := storage.NewEncrypter(cfg.SecretKey)
		brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)

		fixID := "test"
		givenRuntimeState := fixture.FixRuntimeState(fixID, fixID, fixID)
		givenRuntimeState.KymaConfig.Version = fixID
		givenRuntimeState.ClusterConfig.KubernetesVersion = fixID

		svc := brokerStorage.RuntimeStates()

		err = svc.Insert(givenRuntimeState)
		require.NoError(t, err)

		runtimeStates, err := svc.ListByRuntimeID(fixID)
		require.NoError(t, err)
		assert.Len(t, runtimeStates, 1)
		assert.Equal(t, fixID, runtimeStates[0].KymaConfig.Version)
		assert.Equal(t, fixID, runtimeStates[0].ClusterConfig.KubernetesVersion)

		state, err := svc.GetByOperationID(fixID)
		require.NoError(t, err)
		assert.Equal(t, fixID, state.KymaConfig.Version)
		assert.Equal(t, fixID, state.ClusterConfig.KubernetesVersion)
	})

	t.Run("should insert and fetch RuntimeState with Reconciler input", func(t *testing.T) {
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		tablesCleanupFunc, err := storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)
		defer tablesCleanupFunc()

		cipher := storage.NewEncrypter(cfg.SecretKey)
		brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)

		fixRuntimeStateID := uuid.NewString()
		fixRuntimeID := "runtime1"
		fixOperationID := "operation1"
		givenRuntimeState := fixture.FixRuntimeState(fixRuntimeStateID, fixRuntimeID, fixOperationID)
		givenRuntimeState.ClusterSetup = &reconciler.Cluster{
			Cluster: fixRuntimeID,
		}

		storage := brokerStorage.RuntimeStates()

		err = storage.Insert(givenRuntimeState)
		require.NoError(t, err)

		runtimeStates, err := storage.ListByRuntimeID(fixRuntimeID)
		require.NoError(t, err)
		assert.Len(t, runtimeStates, 1)
		assert.Equal(t, fixRuntimeStateID, runtimeStates[0].ID)
		assert.Equal(t, fixRuntimeID, runtimeStates[0].ClusterSetup.Cluster)

		state, err := storage.GetByOperationID(fixOperationID)
		require.NoError(t, err)
		assert.Equal(t, fixRuntimeStateID, state.ID)
		assert.Equal(t, fixRuntimeID, state.ClusterSetup.Cluster)
	})
}
