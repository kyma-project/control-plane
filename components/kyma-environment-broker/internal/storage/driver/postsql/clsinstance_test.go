package postsql_test

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestClsInstance(t *testing.T) {

	ctx := context.Background()

	t.Run("CLS Instances", func(t *testing.T) {
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		tablesCleanupFunc, err := storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)
		defer tablesCleanupFunc()

		cipher := storage.NewEncrypter(cfg.SecretKey)
		brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)

		storage := brokerStorage.CLSInstances()

		globalAccountID := "fake-global-account-id"

		newClsInstance := internal.NewCLSInstance(globalAccountID, "eu", internal.WithReferences("fake-skr-instance-id-1"))
		instanceID := newClsInstance.ID()
		err = storage.Insert(*newClsInstance)
		require.NoError(t, err)
		t.Logf("Inserted the instance: %#v", newClsInstance)

		newClsInstance.AddReference("fake-skr-instance-id-2")
		err = storage.Update(*newClsInstance)
		require.NoError(t, err)
		t.Logf("Referenced the instance %s by the skr %s", instanceID, "fake-skr-instance-id-2")

		gotClsInstance, found, err := storage.FindActiveByGlobalAccountID(globalAccountID)
		require.NoError(t, err)
		require.NotNil(t, gotClsInstance)
		require.True(t, found)
		require.Equal(t, newClsInstance.ID(), gotClsInstance.ID())
		require.Equal(t, newClsInstance.GlobalAccountID(), gotClsInstance.GlobalAccountID())
		require.Equal(t, newClsInstance.Region(), gotClsInstance.Region())
		require.ElementsMatch(t, []string{"fake-skr-instance-id-1", "fake-skr-instance-id-2"}, gotClsInstance.References())
		require.NoError(t, err)
		t.Logf("Found the instance by global id: %#v", gotClsInstance)

		err = gotClsInstance.RemoveReference("fake-skr-instance-id-2")
		require.NoError(t, err)
		err = storage.Update(*gotClsInstance)
		require.NoError(t, err)
		t.Logf("Unreferenced the instance %s by the skr %s", instanceID, "fake-skr-instance-id-2")

		gotClsInstance, found, err = storage.FindByID(instanceID)
		require.NoError(t, err)
		require.NotNil(t, gotClsInstance)
		require.True(t, found)
		require.Equal(t, newClsInstance.ID(), gotClsInstance.ID())
		require.Equal(t, newClsInstance.GlobalAccountID(), gotClsInstance.GlobalAccountID())
		require.Equal(t, newClsInstance.Region(), gotClsInstance.Region())
		require.ElementsMatch(t, []string{"fake-skr-instance-id-1"}, gotClsInstance.References())
		require.NoError(t, err)
		t.Logf("Found the instance by id: %#v", gotClsInstance)

		err = gotClsInstance.RemoveReference("fake-skr-instance-id-1")
		require.NoError(t, err)
		err = storage.Update(*gotClsInstance)
		require.NoError(t, err)
		t.Logf("Unreferenced the instance %s by the skr %s", instanceID, "fake-skr-instance-id-1")

		gotClsInstance, found, err = storage.FindActiveByGlobalAccountID(globalAccountID)
		require.NoError(t, err)
		require.False(t, found)
		require.Nil(t, gotClsInstance)
		t.Logf("Could not find active instance %s", instanceID)

		gotClsInstance, found, err = storage.FindByID(instanceID)
		require.NoError(t, err)
		require.True(t, found)
		require.NotEmpty(t, gotClsInstance.BeingRemovedBy())
		t.Logf("Found inactive active instance %s", instanceID)

		err = storage.Delete(instanceID)
		require.NoError(t, err)
		t.Logf("Removed inactive instance %s", instanceID)

		gotClsInstance, found, err = storage.FindByID(instanceID)
		require.NoError(t, err)
		require.False(t, found)
		require.Nil(t, gotClsInstance)
		t.Logf("Could not find inactive instance %s", instanceID)
	})
}
