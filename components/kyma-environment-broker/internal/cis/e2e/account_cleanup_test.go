//go:build cis
// +build cis

package e2e

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cis"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// go test --tags="cis" -v
func TestSubAccountCleanup(t *testing.T) {
	ctx := context.Background()

	t.Log("ensure docker network")
	cleanupNetwork, err := storage.EnsureTestNetworkForDB(t, ctx)
	require.NoError(t, err)
	defer cleanupNetwork()

	t.Run("CIS 1.0", func(t *testing.T) {
		// Given
		instances := fixInstances()

		t.Log("create image with postgres database")
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		t.Log("initialize database by creating instances table")
		err = initTestDBInstancesTables(t, cfg.ConnectionURL())
		require.NoError(t, err)

		t.Log("create storage manager")
		cipher := storage.NewEncrypter(cfg.SecretKey)
		storageManager, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
		require.NoError(t, err)

		t.Log("fill instances table")
		for _, instance := range instances {
			err := storageManager.Instances().Insert(instance)
			require.NoError(t, err)
		}

		t.Log("create CIS fake server")
		testServer := fixHTTPServer(t)
		defer testServer.Close()

		t.Log("create CIS client")
		client := cis.NewClientVer1(context.TODO(), cis.Config{
			EventServiceURL: testServer.URL,
			PageSize:        "10",
		}, logger.NewLogDummy())
		client.SetHttpClient(testServer.Client())

		t.Log("create broker client mock and assert execution deprovisioning for first 30 instances")
		brokerClient := NewFakeBrokerClient(storageManager.Instances())

		t.Log("create subaccount cleanup service")
		sacs := cis.NewSubAccountCleanupService(client, brokerClient, storageManager.Instances(), logger.NewLogDummy())

		// When
		err = sacs.Run()

		// Then
		require.NoError(t, err)

		amount, err := storageManager.Instances().GetNumberOfInstancesForGlobalAccountID(globalAccountID)
		require.NoError(t, err)
		require.Equal(t, 10, amount)
	})

	t.Run("CIS 2.0", func(t *testing.T) {
		// Given
		instances := fixInstances()

		t.Log("create image with postgres database")
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_2")
		require.NoError(t, err)
		defer containerCleanupFunc()

		t.Log("initialize database by creating instances table")
		err = initTestDBInstancesTables(t, cfg.ConnectionURL())
		require.NoError(t, err)

		t.Log("create storage manager")
		cipher := storage.NewEncrypter(cfg.SecretKey)
		storageManager, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
		require.NoError(t, err)

		t.Log("fill instances table")
		for _, instance := range instances {
			err := storageManager.Instances().Insert(instance)
			require.NoError(t, err)
		}

		t.Log("create CIS fake server")
		testServer := fixHTTPServer(t)
		defer testServer.Close()

		t.Log("create CIS client")
		client := cis.NewClient(context.TODO(), cis.Config{
			EventServiceURL: testServer.URL,
			PageSize:        "10",
		}, logger.NewLogDummy())
		client.SetHttpClient(testServer.Client())

		t.Log("create broker client mock and assert execution deprovisioning for first 30 instances")
		brokerClient := NewFakeBrokerClient(storageManager.Instances())

		t.Log("create subaccount cleanup service")
		sacs := cis.NewSubAccountCleanupService(client, brokerClient, storageManager.Instances(), logger.NewLogDummy())

		// When
		err = sacs.Run()

		// Then
		require.NoError(t, err)

		amount, err := storageManager.Instances().GetNumberOfInstancesForGlobalAccountID(globalAccountID)
		require.NoError(t, err)
		require.Equal(t, 10, amount)
	})
}
