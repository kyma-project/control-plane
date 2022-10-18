package postsql_test

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenericRead(t *testing.T) {

	ctx := context.Background()

	t.Run("Should Read Instance by Id", func(t *testing.T) {
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t.Logf, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		cleanupNetwork, err := storage.SetupTestNetworkForDB(ctx)
		if err != nil {
			log.Fatal(err)
		}
		defer cleanupNetwork()

		tablesCleanupFunc, err := storage.InitTestDBTablesWithPath(t, cfg.ConnectionURL(), "./../../../../schema-migrator/migrations/kyma-environment-broker/")
		require.NoError(t, err)
		defer tablesCleanupFunc()

		logger := logrus.StandardLogger()

		connection, err := postsql.InitializeDatabase(cfg.ConnectionURL(), 10, logger)
		assert.NoError(t, err)

		defer connection.Close()

		instanceId := "instance_id"

		session := postsql.NewFactory(connection).
			NewWriteSession()

		instance := dbmodel.InstanceDTO{
			InstanceID:                  instanceId,
			RuntimeID:                   "RuntimeID",
			GlobalAccountID:             "GlobalAccount",
			SubscriptionGlobalAccountID: "SubsGlobalAccount",
			SubAccountID:                "SubAccountID",
			ServiceID:                   "ServiceID",
			ServiceName:                 "ServiceName",
			ServicePlanID:               "ServicePlanID",
			ServicePlanName:             "ServicePlanName",

			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			DeletedAt: time.Now(),
		}
		err = session.InsertInstance(instance)
		assert.NoError(t, err)

		readInstances := postsql.GenericRead[dbmodel.InstanceDTO]{
			Session:   connection.NewSession(nil),
			IdName:    instanceId,
			TableName: postsql.InstancesTableName,
			NewItem: func() dbmodel.InstanceDTO {
				return dbmodel.InstanceDTO{}
			},
		}
		dbInstance, err := readInstances.GetByID(instanceId)

		assert.NoError(t, err)
		assert.NotNil(t, dbInstance)
		assert.Equal(t, instance.InstanceID, dbInstance.InstanceID)
		assert.Equal(t, instance.RuntimeID, dbInstance.RuntimeID)
		assert.Equal(t, instance.GlobalAccountID, dbInstance.GlobalAccountID)
		assert.Equal(t, instance.SubscriptionGlobalAccountID, dbInstance.SubscriptionGlobalAccountID)
		assert.Equal(t, instance.SubAccountID, dbInstance.SubAccountID)
		assert.Equal(t, instance.ServiceID, dbInstance.ServiceID)
		assert.Equal(t, instance.ServiceName, dbInstance.ServiceName)
		assert.Equal(t, instance.ServicePlanID, dbInstance.ServicePlanID)
		assert.Equal(t, instance.ServicePlanName, dbInstance.ServicePlanName)
	})
}
