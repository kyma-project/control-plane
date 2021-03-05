package postsql_test

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLMSTenant(t *testing.T) {
	ctx := context.Background()
	cleanupNetwork, err := storage.EnsureTestNetworkForDB(t, ctx)
	require.NoError(t, err)
	defer cleanupNetwork()

	t.Run("LMS Tenants", func(t *testing.T) {
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		lmsTenant := internal.LMSTenant{
			ID:     "tenant-001",
			Region: "na",
			Name:   "some-company",
		}
		err = storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)

		cipher := storage.NewEncrypter(cfg.SecretKey)
		brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
		svc := brokerStorage.LMSTenants()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)

		// when
		err = svc.InsertTenant(lmsTenant)
		require.NoError(t, err)
		gotTenant, found, err := svc.FindTenantByName("some-company", "na")
		_, differentRegionExists, drErr := svc.FindTenantByName("some-company", "us")
		_, differentNameExists, dnErr := svc.FindTenantByName("some-company1", "na")

		// then
		assert.Equal(t, lmsTenant.Name, gotTenant.Name)
		assert.True(t, found)
		assert.NoError(t, err)
		assert.False(t, differentRegionExists)
		assert.NoError(t, drErr)
		assert.False(t, differentNameExists)
		assert.NoError(t, dnErr)
	})
}
