package postsql_test

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrchestration(t *testing.T) {

	ctx := context.Background()

	t.Run("Orchestrations", func(t *testing.T) {
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t.Logf, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		tablesCleanupFunc, err := storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)
		defer tablesCleanupFunc()

		cipher := storage.NewEncrypter(cfg.SecretKey)
		brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)

		givenOrchestration := fixture.FixOrchestration("test")
		givenOrchestration.Type = orchestration.UpgradeKymaOrchestration
		givenOrchestration.State = "test"
		givenOrchestration.Description = "test"
		givenOrchestration.Parameters.DryRun = true

		svc := brokerStorage.Orchestrations()

		err = svc.Insert(givenOrchestration)
		require.NoError(t, err)

		// when
		gotOrchestration, err := svc.GetByID("test")
		require.NoError(t, err)
		assert.Equal(t, givenOrchestration.Parameters, gotOrchestration.Parameters)
		assert.Equal(t, orchestration.UpgradeKymaOrchestration, gotOrchestration.Type)

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

		l, c, tc, err := svc.List(dbmodel.OrchestrationFilter{States: []string{"test"}, Types: []string{string(orchestration.UpgradeKymaOrchestration)}})
		require.NoError(t, err)
		assert.Len(t, l, 1)
		assert.Equal(t, 1, c)
		assert.Equal(t, 1, tc)
	})
}
