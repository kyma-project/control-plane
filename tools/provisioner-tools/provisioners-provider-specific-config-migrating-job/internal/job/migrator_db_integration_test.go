package job

import (
	"context"
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/persistence/dbconnection"
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/testutils"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProviderConfigMigrator(t *testing.T) {
	ctx := context.Background()

	cleanupNetwork, err := testutils.EnsureTestNetworkForDB(t, ctx)
	require.NoError(t, err)
	defer cleanupNetwork()

	containerCleanupFunc, connString, err := testutils.InitTestDBContainer(t, ctx, "postgres_database_2")
	require.NoError(t, err)
	defer containerCleanupFunc()

	connection, err := dbconnection.InitializeDatabaseConnection(connString, 5)
	require.NoError(t, err)
	require.NotNil(t, connection)
	defer testutils.CloseDatabase(t, connection)

	err = testutils.SetupSchema(connection, testutils.SchemaFilePath)
	require.NoError(t, err)

}
