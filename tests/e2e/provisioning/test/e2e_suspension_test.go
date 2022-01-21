package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_E2E_Suspension(t *testing.T) {
	ts := newTestSuite(t)
	if ts.IsDummyTest {
		return
	}
	if ts.IsUpgradeTest {
		return
	}
	if ts.IsCleanupPhase {
		ts.Cleanup()
		return
	}

	configMap := ts.testConfigMap()

	ts.log.Info("Starting e2e suspension test")
	operationID, err := ts.brokerClient.ProvisionRuntime("")
	require.NoError(t, err)

	ts.log.Infof("Creating config map %s with test data", ts.ConfigName)
	err = ts.configMapClient.Create(configMap)
	require.NoError(t, err)

	err = ts.brokerClient.AwaitOperationSucceeded(operationID, ts.ProvisionTimeout)
	require.NoError(t, err)

	dashboardURL, err := ts.brokerClient.FetchDashboardURL()
	require.NoError(t, err)

	ts.log.Infof("Updating config map %s with dashboardUrl", ts.ConfigName)
	configMap.Data[dashboardUrlKey] = dashboardURL
	err = ts.configMapClient.Update(configMap)
	require.NoError(t, err)

	err = ts.dashboardChecker.AssertRedirectedToBusola(dashboardURL, ts.BusolaURL)
	assert.NoError(t, err)

	err = ts.brokerClient.SuspendRuntime()
	assert.NoError(t, err)

	err = ts.brokerClient.AwaitOperationSucceeded("", ts.DeprovisionTimeout)
	require.NoError(t, err)

	err = ts.brokerClient.UnsuspendRuntime()
	assert.NoError(t, err)

	err = ts.brokerClient.AwaitOperationSucceeded("", ts.ProvisionTimeout)
	require.NoError(t, err)

	ts.log.Info("Fetching runtime's kubeconfig")
	config, err := ts.runtimeClient.FetchRuntimeConfig()
	require.NoError(t, err)

	ts.log.Infof("Creating a secret %s with test data", ts.ConfigName)
	err = ts.secretClient.Create(ts.testSecret(config))
	require.NoError(t, err)

	err = ts.dashboardChecker.AssertRedirectedToBusola(dashboardURL, ts.BusolaURL)
	assert.NoError(t, err)
}
