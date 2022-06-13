package main

//TODO: intentionally disabled, depends on https://github.com/kyma-project/control-plane/pull/1563
/*
func TestClusterUpgrade_UpgradeAfterMigrationWithNetworkPolicy(t *testing.T) {
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
		"sm_platform_credentials": {
			"url": "https://sm.url",
			"credentials": {
			"basic": {
					"username":"smUsername",
					"password":"smPassword"
	  			}
			}
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

	// migrate svcat
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"context": {
		"globalaccount_id": "g-account-id",
		"subaccount_id": "sub-id",
		"user_id": "john.smith@email.com",
		"sm_operator_credentials": {
			"clientid": "testClientID",
			"clientsecret": "testClientSecret",
			"sm_url": "https://service-manager.kyma.com",
			"url": "https://test.auth.com",
			"xsappname": "testXsappname"
		},
		"isMigration": true,
		"license_type": "CUSTOMER"
	}
}`)

	// make sure migration finished
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByReconcilerBoth(updateOperationID)
	suite.WaitForOperationState(updateOperationID, domain.Succeeded)

	// ensure component list after update is correct
	i, err := suite.db.Instances().GetByID(id)
	assert.NoError(t, err, "getting instance after update")
	assert.True(t, i.InstanceDetails.SCMigrationTriggered, "instance SCMigrationTriggered after update")
	rsu1, err := suite.db.RuntimeStates().GetLatestWithReconcilerInputByRuntimeID(i.RuntimeID)
	assert.NoError(t, err, "getting runtime after update")
	assert.Equal(t, updateOperationID, rsu1.OperationID, "runtime state update operation ID")
	assert.ElementsMatch(t, componentNames(rsu1.ClusterSetup.KymaConfig.Components), []string{"ory", "monitoring", "btp-operator"})

	// ensure license type is persisted but network filter not enabled
	instance2 := suite.GetInstance(id)
	suite.AssertDisabledNetworkFilterRuntimeState(i.RuntimeID, updateOperationID, nil)
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
	opResponse := suite.CallAPI("GET", fmt.Sprintf("orchestrations/%s/operations", oID), "")
	upgradeClusterOperationID, err := suite.DecodeLastUpgradeKymaOperationIDFromOrchestration(opResponse)
	require.NoError(t, err)

	suite.WaitForOperationState(upgradeClusterOperationID, domain.InProgress)
	suite.FinishUpgradeClusterOperationByProvisioner(upgradeClusterOperationID)
	suite.WaitForOperationState(upgradeClusterOperationID, domain.Succeeded)

	_, err = suite.db.Operations().GetUpgradeClusterOperationByID(upgradeClusterOperationID)
	require.NoError(t, err)

	// ensure component list after upgrade didn't get changed
	i, err = suite.db.Instances().GetByID(id)
	assert.NoError(t, err, "getting instance after upgrade")
	assert.True(t, i.InstanceDetails.SCMigrationTriggered, "instance SCMigrationTriggered after upgrade")
	rsu2, err := suite.db.RuntimeStates().GetLatestWithReconcilerInputByRuntimeID(i.RuntimeID)
	assert.NoError(t, err, "getting runtime after upgrade")
	assert.NotEqual(t, rsu1.ID, rsu2.ID, "runtime_state ID from update should differ runtime_state ID from upgrade")
	assert.Equal(t, upgradeClusterOperationID, rsu2.OperationID, "runtime state upgrade operation ID")
	assert.ElementsMatch(t, componentNames(rsu2.ClusterSetup.KymaConfig.Components), []string{"ory", "monitoring", "btp-operator"})

	assert.Equal(t, rsu1.ClusterConfig.Name, rsu2.ClusterConfig.Name)

	// ensure license type still persisted and network filter disabled during upgrade
	fmt.Println("DEBUG_DELETE network check")
	states, _ := suite.db.RuntimeStates().ListByRuntimeID(i.RuntimeID)
	fmt.Println("DEBUG_DELETE list rs")
	sort.Slice(states, func(i, j int) bool { return states[i].CreatedAt.Before(states[j].CreatedAt) })
	opIds := make(map[string]bool)
	for i, s := range states {
		fmt.Println("DEBUG_DELETE rs", i, s.ID, s.OperationID, s.ClusterConfig.ShootNetworkingFilterDisabled, "[", s.ClusterConfig.Provider, "]", s.ClusterSetup != nil)
		opIds[s.OperationID] = true
	}
	var o []string
	for op, _ := range opIds {
		o = append(o, op)
	}
	fmt.Println("DEBUG_DELETE getting operations for", o)
	ops, _ := suite.db.Operations().GetOperationsForIDs(o)
	sort.Slice(ops, func(i, j int) bool { return ops[i].CreatedAt.Before(ops[j].CreatedAt) })
	for i, op := range ops {
		fmt.Println("DEBUG_DELETE op", i, op.ID, op.Type, op.ProvisioningParameters.ErsContext.LicenseType)
	}

	disabled := true
	suite.AssertDisabledNetworkFilterRuntimeState(i.RuntimeID, upgradeClusterOperationID, &disabled)
	assert.Equal(suite.t, "CUSTOMER", *instance2.Parameters.ErsContext.LicenseType)
}
*/
