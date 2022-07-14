package postsql_test

import (
	"context"
	"log"
	"testing"

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

		session := connection.NewSession(nil)
		defer session.Close()

		tx, err := session.Begin()
		assert.NoError(t, err)

		testServiceName := "Test Service Name"

		_, err = session.InsertInto(postsql.InstancesTableName).
			Pair("instance_id", instanceId).
			Pair("runtime_id", "runtime_id").
			Pair("global_account_id", "global_account_id").
			Pair("service_id", "service_id").
			Pair("service_plan_id", "service_plan_id").
			Pair("dashboard_url", "dashboard_url").
			Pair("service_name", testServiceName).
			Pair("provisioning_parameters", "provisioning_parameters").
			Exec()
		assert.NoError(t, err)

		err = tx.Commit()
		assert.NoError(t, err)

		readInstances := postsql.GenericRead[dbmodel.InstanceDTO]{
			Session:   connection.NewSession(nil),
			IdName:    instanceId,
			TableName: postsql.InstancesTableName,
			NewItem: func() dbmodel.InstanceDTO {
				return dbmodel.InstanceDTO{}
			},
		}
		instance, err := readInstances.GetInstanceByID(instanceId)

		assert.NoError(t, err)
		assert.NotNil(t, instance)
		assert.Equal(t, testServiceName, instance.ServiceName)
	})
}
