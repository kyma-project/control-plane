package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusterUpgrade_UpgradeAfterUpdateWithNetworkPolicy(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	mockBTPOperatorClusterID()
	defer suite.TearDown()
	id := "InstanceID-UpgradeAfterMigration"

	// provision Kyma 2.0
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
	"context": {
		"sm_operator_credentials": {
			"clientid": "testClientID",
			"clientsecret": "testClientSecret",
			"sm_url": "https://service-manager.kyma.com",
			"url": "https://test.auth.com",
			"xsappname": "testXsappname"
		},
		"globalaccount_id": "g-account-id",
		"subaccount_id": "sub-id",
		"user_id": "john.smith@email.com"
	},
	"parameters": {
		"name": "testing-cluster"
	}
}`)
	opID := suite.DecodeOperationID(resp)
	suite.processReconcilingByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// provide license_type
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"context": {
		"license_type": "CUSTOMER"
	}
}`)

	// finish the update operation
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(updateOperationID)
	suite.WaitForOperationState(updateOperationID, domain.Succeeded)
	i, err := suite.db.Instances().GetByID(id)
	rsu1, err := suite.db.RuntimeStates().GetLatestWithReconcilerInputByRuntimeID(i.RuntimeID)

	// ensure license type is persisted and network filter enabled
	instance2 := suite.GetInstance(id)
	enabled := true
	suite.AssertDisabledNetworkFilterRuntimeState(i.RuntimeID, updateOperationID, &enabled)
	assert.Equal(suite.t, "CUSTOMER", *instance2.Parameters.ErsContext.LicenseType)

	// run upgrade
	orchestrationResp := suite.CallAPI("POST", "upgrade/cluster", `
{
	"strategy": {
		"type": "parallel",
		"schedule": "immediate",
		"parallel": {
			"workers": 1
		}
	},
	"dryRun": false,
	"targets": {
		"include": [
			{
				"subAccount": "sub-id"
			}
		]
	},
	"kubernetes": {
		"kubernetesVersion": "1.25.0"
	}
}`)
	oID := suite.DecodeOrchestrationID(orchestrationResp)
	upgradeClusterOperationID, err := suite.DecodeLastUpgradeClusterOperationIDFromOrchestration(oID)
	require.NoError(t, err)

	suite.WaitForOperationState(upgradeClusterOperationID, domain.InProgress)
	suite.FinishUpgradeClusterOperationByProvisioner(upgradeClusterOperationID)
	suite.WaitForOperationState(upgradeClusterOperationID, domain.Succeeded)

	_, err = suite.db.Operations().GetUpgradeClusterOperationByID(upgradeClusterOperationID)
	require.NoError(t, err)

	// ensure component list after upgrade didn't get changed
	i, err = suite.db.Instances().GetByID(id)
	assert.NoError(t, err, "getting instance after upgrade")
	rsu2, err := suite.db.RuntimeStates().GetLatestWithReconcilerInputByRuntimeID(i.RuntimeID)
	assert.NoError(t, err, "getting runtime after upgrade")
	assert.Equal(t, rsu1.ClusterConfig.Name, rsu2.ClusterConfig.Name)

	// ensure license type still persisted and network filter still disabled after upgrade
	disabled := true
	suite.AssertDisabledNetworkFilterRuntimeState(i.RuntimeID, upgradeClusterOperationID, &disabled)
	assert.Equal(suite.t, "CUSTOMER", *instance2.Parameters.ErsContext.LicenseType)
}
