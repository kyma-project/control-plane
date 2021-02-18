package migrations_test

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/migrations"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestInstanceParametersMigrationRollback_Migrate(t *testing.T) {
	t.Run("should decrypt Instance parameters", func(t *testing.T) {
		ctx := context.Background()

		cleanupNetwork, err := storage.EnsureTestNetworkForDB(t, ctx)
		require.NoError(t, err)
		defer cleanupNetwork()

		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		err = storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)

		cipher := storage.NewEncrypter(cfg.SecretKey)
		s, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
		require.NoError(t, err)

		// given
		testValue := "test"

		// insert encrypts credentials
		err = s.Instances().Insert(internal.Instance{
			InstanceID: testValue,
			Parameters: internal.ProvisioningParameters{
				ErsContext: internal.ERSContext{
					ServiceManager: &internal.ServiceManagerEntryDTO{
						Credentials: internal.ServiceManagerCredentials{
							BasicAuth: internal.ServiceManagerBasicAuth{
								Username: testValue,
								Password: testValue,
							},
						},
					},
				},
			},
		})
		require.NoError(t, err)

		// when
		err = migrations.NewInstanceParametersMigrationRollback(s.Instances(), logrus.New()).Migrate()
		require.NoError(t, err)

		// then
		i, _, _, err := s.Instances().ListWithoutDecryption(dbmodel.InstanceFilter{})
		require.NoError(t, err)
		assert.Equal(t, testValue, i[0].Parameters.ErsContext.ServiceManager.Credentials.BasicAuth.Username)
		assert.Equal(t, testValue, i[0].Parameters.ErsContext.ServiceManager.Credentials.BasicAuth.Password)
	})
}
