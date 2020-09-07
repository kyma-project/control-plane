package test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_E2E_Upgrade(t *testing.T) {
	ts := newTestSuite(t)
	if ts.IsDummyTest {
		return
	}
	if !ts.IsUpgradeTest {
		t.SkipNow()
	}
	if ts.IsCleanupPhase {
		ts.Cleanup()
		return
	}
	configMap := ts.testConfigMap()

	operationID, err := ts.brokerClient.ProvisionRuntime("")
	require.NoError(t, err)

	ts.log.Infof("Creating config map %s with test data", ts.ConfigName)
	err = ts.configMapClient.Create(configMap)
	require.NoError(t, err)

	err = ts.brokerClient.AwaitOperationSucceeded(operationID, ts.ProvisionTimeout)
	require.NoError(t, err)

	ts.log.Info("Fetching runtime's kubeconfig")
	config, err := ts.runtimeClient.FetchRuntimeConfig()
	require.NoError(t, err)

	ts.log.Infof("Creating a secret %s with test data", ts.ConfigName)
	err = ts.secretClient.Create(ts.testSecret(config))
	require.NoError(t, err)

	ts.log.Infof("Fetch runtimeID from CLD endpoint based on instanceID: %s", ts.InstanceID)
	runtimeID, err := ts.upgradeSuite.upgradeClient.FetchRuntimeID(ts.InstanceID)
	require.NoError(t, err)

	ts.log.Infof("Starting upgrade runtime with ID: %s", runtimeID)
	orchestrationID, err := ts.upgradeSuite.upgradeClient.UpgradeRuntime(runtimeID)
	require.NoError(t, err, "failed to upgrade Runtime")

	ts.log.Infof("Waiting for upgrade to finish for orchestrationID: %s", orchestrationID)
	err = ts.upgradeSuite.upgradeClient.AwaitOperationFinished(orchestrationID, ts.upgradeSuite.UpgradeTimeout)
	require.NoError(t, err, "error waiting for upgrade to finish")

	ts.log.Info("Test completed successfully")
}
