package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/stretchr/testify/assert"
)

func TestUpgrade(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	suite.EnableDumpingProvisionerRequests()
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		`{
				   "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				   "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
				   "context": {
					   "sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					   },
					   "globalaccount_id": "g-account-id",
					   "subaccount_id": "sub-id",
					   "user_id": "john.smith@email.com"
				   },
					"parameters": {
						"name": "testing-cluster",
						"oidc": {
							"clientid": "id-initial",
							"signingalgs": ["xxx"],
                            "issuerurl": "https://issuer.url.com"
						}
			}
   }`)
	opId := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opId)

	provOp, _ := suite.db.Provisioning().GetProvisioningOperationByID(opId)
	runtimeId := provOp.RuntimeID

	// when
	// kyma upgrade - create orchestration:

	resp = suite.CallAPI("POST", "upgrade/kyma",
		fmt.Sprintf(`{
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
	}`, runtimeId))
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	orchestrationId := suite.DecodeOrchestrationID(resp)

	suite.Log(orchestrationId)

	suite.processUpgradeKymaByInstanceID(provOp.InstanceID)

	suite.WaitForOrchestrationState(orchestrationId, orchestration.Succeeded)

	suite.AssertUpgradeKyma(provOp.ID, runtimeId)
}
