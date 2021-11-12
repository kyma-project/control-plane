package main

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestKymaUpgrade_UpgradeTo2(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// provision Kyma 1.x
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
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
	suite.processProvisioningByOperationID(opID)

	// when
	orchestrationResp := suite.CallAPI("POST", "upgrade/kyma",
		`{
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
					"kyma": {
						"version": "2.0.0-rc4"
					}
				}`)
	oID := suite.DecodeOrchestrationID(orchestrationResp)


	suite.AssertReconcilerStartedReconcilingWhenUpgrading(iid)

	opResponse := suite.CallAPI("GET", fmt.Sprintf("orchestrations/%s/operations", oID), "")
	upgradeKymaOperationID, err := suite.DecodeLastUpgradeKymaOperationIDFromOrchestration(opResponse)
	require.NoError(t, err)

	fmt.Println(upgradeKymaOperationID)

	suite.FinishUpgradeKymaOperationByReconciler(upgradeKymaOperationID)
	suite.Log(fmt.Sprintf("orchestration created with id %q", oID))
	//TODO: assert no upgrade calls went to provisioner, reconciler got proper configuration
}
