package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestClsPostgres(t *testing.T) {
	ctx := context.Background()

	cleanupNetwork, err := storage.EnsureTestNetworkForDB(t, ctx)
	require.NoError(t, err)
	defer cleanupNetwork()

	t.Run("CLS", func(t *testing.T) {
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		err = storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)

		cipher := storage.NewEncrypter(cfg.SecretKey)
		brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
		storage := brokerStorage.CLSInstances()
		require.NotNil(t, brokerStorage)
		require.NoError(t, err)

		globalAccountID := "fake-global-account-id"

		instanceID := "fake-id"
		newClsInstance := internal.CLSInstance{
			ID:                       instanceID,
			GlobalAccountID:          globalAccountID,
			Region:                   "eu",
			CreatedAt:                time.Now().UTC(),
			ReferencedSKRInstanceIDs: []string{"fake-skr-instance-id-1"},
		}
		err = storage.Insert(newClsInstance)
		require.NoError(t, err)
		t.Logf("Inserted the instance: %#v", newClsInstance)

		skrID := "fake-skr-instance-id-2"
		err = storage.Reference(newClsInstance.Version, instanceID, skrID)
		require.NoError(t, err)
		t.Logf("Referenced the instance %s by the skr %s", instanceID, skrID)

		err = storage.Reference(newClsInstance.Version, instanceID, "fake-skr-instance-id-3")
		require.Error(t, err)
		t.Logf("Failed to reference the instance %s by the skr %s: %s", instanceID, skrID, err)

		gotClsInstance, found, err := storage.FindActiveByGlobalAccountID("fake-global-account-id")
		require.NoError(t, err)
		require.NotNil(t, gotClsInstance)
		require.True(t, found)
		require.Equal(t, newClsInstance.ID, gotClsInstance.ID)
		require.Equal(t, newClsInstance.GlobalAccountID, gotClsInstance.GlobalAccountID)
		require.Equal(t, newClsInstance.Region, gotClsInstance.Region)
		require.ElementsMatch(t, []string{"fake-skr-instance-id-1", "fake-skr-instance-id-2"}, gotClsInstance.ReferencedSKRInstanceIDs)
		require.NoError(t, err)
		t.Logf("Found the instance by global id: %#v", gotClsInstance)

		skrID = "fake-skr-instance-id-2"
		err = storage.Unreference(gotClsInstance.Version, instanceID, skrID)
		require.NoError(t, err)
		t.Logf("Uneferenced the instance %s by the skr %s", instanceID, skrID)

		gotClsInstance, _, err = storage.FindByID(newClsInstance.ID)
		require.NoError(t, err)
		require.Equal(t, newClsInstance.ID, gotClsInstance.ID)
		require.Equal(t, newClsInstance.GlobalAccountID, gotClsInstance.GlobalAccountID)
		require.Equal(t, newClsInstance.Region, gotClsInstance.Region)
		require.ElementsMatch(t, []string{"fake-skr-instance-id-1"}, gotClsInstance.ReferencedSKRInstanceIDs)
		require.NoError(t, err)
		t.Logf("Found the instance by id: %#v", gotClsInstance)

		err = storage.MarkAsBeingRemoved(gotClsInstance.Version, instanceID, "fake-skr-instance-id-1")
		require.NoError(t, err)
		t.Logf("Marked the instance %s as being removed", instanceID)

		gotClsInstance, found, err = storage.FindActiveByGlobalAccountID("fake-global-account-id")
		require.NoError(t, err)
		require.False(t, found)
		require.Nil(t, gotClsInstance)
		t.Logf("Could not find active instance %s", instanceID)

		gotClsInstance, found, err = storage.FindByID(instanceID)
		require.NoError(t, err)
		require.True(t, found)
		require.NotNil(t, gotClsInstance)
		require.NotEmpty(t, gotClsInstance.ReferencedSKRInstanceIDs)
		t.Logf("Found inactive active instance %s", instanceID)

		err = storage.Remove(instanceID)
		require.NoError(t, err)
		t.Logf("Removed inactive instance %s", instanceID)

		gotClsInstance, found, err = storage.FindByID(instanceID)
		require.NoError(t, err)
		require.False(t, found)
		require.Nil(t, gotClsInstance)
		t.Logf("Could not find inactive instance %s", instanceID)
	})
}
