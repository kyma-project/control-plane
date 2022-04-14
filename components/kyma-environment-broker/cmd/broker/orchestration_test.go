package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestKymaUpgrade_OneRuntimeHappyPath(t *testing.T) {
	/*
		There are two runtimes (kyma 2.0), then trigger orchestration for one of them (upgrade to kyma 2.1).
		Check, if the upgrade was processed only for that one.
	*/
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid1 := uuid.New().String()
	iid2 := uuid.New().String()

	// given
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid1),
		`{
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
						"subaccount_id": "sub-id1",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
                        "kymaVersion": "2.0"
					}
		}`)
	provisioningOperation1 := suite.DecodeOperationID(resp)
	suite.processReconcilingByOperationID(provisioningOperation1)

	// given
	resp = suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid2),
		`{
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
						"subaccount_id": "sub-id2",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
                        "kymaVersion": "2.0"
					}
		}`)
	provisioningOperation2 := suite.DecodeOperationID(resp)
	suite.processReconcilingByOperationID(provisioningOperation2)

	suite.WaitForOperationState(provisioningOperation1, domain.Succeeded)
	suite.WaitForOperationState(provisioningOperation2, domain.Succeeded)
	runtimeID1 := suite.GetInstance(iid1).RuntimeID

	// run upgrade
	orchestrationResp := suite.CallAPI("POST", "upgrade/kyma", fmt.Sprintf(`
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
				"runtimeID": "%s"
			}
		]
	},
	"kyma": {
		"version": "2.1"
	}
}`, runtimeID1))
	oID := suite.DecodeOrchestrationID(orchestrationResp)
	suite.AssertReconcilerStartedReconcilingWhenUpgrading(iid1)

	opResponse := suite.CallAPI("GET", fmt.Sprintf("orchestrations/%s/operations", oID), "")
	upgradeKymaOperationIDs, err := suite.DecodeOperationIDsFromOrchestration(opResponse)
	require.NoError(t, err)
	// we expect only one upgrade Kyma operation
	assert.Len(t, upgradeKymaOperationIDs, 1)

	// when
	suite.FinishUpgradeKymaOperationByReconciler(upgradeKymaOperationIDs[0])

	// then
	clusterConfig, err := suite.reconcilerClient.LastClusterConfig(runtimeID1)
	require.NoError(t, err)
	assert.Equal(t, "2.1", clusterConfig.KymaConfig.Version)
}

func TestClusterUpgrade_OneRuntimeHappyPath(t *testing.T) {
	/*
		There are two runtimes (kyma 2.0), then trigger orchestration for one of them (upgrade to kyma 2.1).
		Check, if the upgrade was processed only for that one.
	*/
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid1 := uuid.New().String()
	iid2 := uuid.New().String()

	// given
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid1),
		`{
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
						"subaccount_id": "sub-id1",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
                        "kymaVersion": "2.0"
					}
		}`)
	provisioningOperation1 := suite.DecodeOperationID(resp)
	suite.processReconcilingByOperationID(provisioningOperation1)

	// given
	resp = suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid2),
		`{
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
						"subaccount_id": "sub-id2",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster",
                        "kymaVersion": "2.0"
					}
		}`)
	provisioningOperation2 := suite.DecodeOperationID(resp)
	suite.processReconcilingByOperationID(provisioningOperation2)

	suite.WaitForOperationState(provisioningOperation1, domain.Succeeded)
	suite.WaitForOperationState(provisioningOperation2, domain.Succeeded)
	runtimeID1 := suite.GetInstance(iid1).RuntimeID

	// run upgrade
	orchestrationResp := suite.CallAPI("POST", "upgrade/cluster", fmt.Sprintf(`
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
				"runtimeID": "%s"
			}
		]
	}
}`, runtimeID1))
	oID := suite.DecodeOrchestrationID(orchestrationResp)
	suite.AssertReconcilerStartedReconcilingWhenUpgrading(iid1)

	opResponse := suite.CallAPI("GET", fmt.Sprintf("orchestrations/%s/operations", oID), "")
	upgradeKymaOperationIDs, err := suite.DecodeOperationIDsFromOrchestration(opResponse)
	require.NoError(t, err)
	// we expect only one upgrade Kyma operation
	assert.Len(t, upgradeKymaOperationIDs, 1)

	// when
	suite.FinishUpgradeKymaOperationByReconciler(upgradeKymaOperationIDs[0])

	// then
	clusterConfig, err := suite.reconcilerClient.LastClusterConfig(runtimeID1)
	require.NoError(t, err)
	assert.Equal(t, "2.1", clusterConfig.KymaConfig.Version)
}
