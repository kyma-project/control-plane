package postsql_test

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialization(t *testing.T) {

	ctx := context.Background()

	t.Run("Should initialize database when schema not applied", func(t *testing.T) {
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t.Logf, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		// when
		connection, err := postsql.InitializeDatabase(cfg.ConnectionURL(), 1, logrus.New())
		require.NoError(t, err)
		require.NotNil(t, connection)

		defer storage.CloseDatabase(t, connection)

		// then
		assert.NoError(t, err)
	})

	t.Run("Should return error when failed to connect to the database", func(t *testing.T) {
		containerCleanupFunc, _, err := storage.InitTestDBContainer(t.Logf, ctx, "test_DB_3")
		require.NoError(t, err)
		defer containerCleanupFunc()

		// given
		connString := "bad connection string"

		// when
		connection, err := postsql.InitializeDatabase(connString, 1, logrus.New())

		// then
		assert.Error(t, err)
		assert.Nil(t, connection)
	})
}
