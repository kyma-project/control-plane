package postsql_test

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrchestration(t *testing.T) {

	if testsRanInSuite {
		t.Skip("TestOrchestration already ran in suite")
	}

	ctx := context.Background()
	cleanupNetwork, err := storage.EnsureTestNetworkForDB(t, ctx)
	require.NoError(t, err)
	defer cleanupNetwork()

	t.Run("Orchestrations", func(t *testing.T) {
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		givenOrchestration := fixture.FixOrchestration("test")
		givenOrchestration.State = "test"
		givenOrchestration.Description = "test"
		givenOrchestration.Parameters.DryRun = true

		tablesCleanupFunc, err := storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)
		defer tablesCleanupFunc()

		cipher := storage.NewEncrypter(cfg.SecretKey)
		brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
		require.NoError(t, err)

		svc := brokerStorage.Orchestrations()

		err = svc.Insert(givenOrchestration)
		require.NoError(t, err)

		// when
		gotOrchestration, err := svc.GetByID("test")
		require.NoError(t, err)
		assert.Equal(t, givenOrchestration.Parameters, gotOrchestration.Parameters)

		gotOrchestration.Description = "new modified description 1"
		err = svc.Update(givenOrchestration)
		require.NoError(t, err)

		err = svc.Insert(givenOrchestration)
		assertError(t, dberr.CodeAlreadyExists, err)

		l, count, totalCount, err := svc.List(dbmodel.OrchestrationFilter{PageSize: 10, Page: 1})
		require.NoError(t, err)
		assert.Len(t, l, 1)
		assert.Equal(t, 1, count)
		assert.Equal(t, 1, totalCount)

		l, err = svc.ListByState("test")
		require.NoError(t, err)
		assert.Len(t, l, 1)
	})
}
