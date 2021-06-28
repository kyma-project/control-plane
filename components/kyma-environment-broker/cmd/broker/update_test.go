package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestUpdate(t *testing.T) {
	suite := NewUpdateSuite(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := procressProvisioning(suite, iid)
	// provisioning done, let's start an update

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
		"parameters": {
			"oidc": {
				"clientID": "id-ooo",
				"signingAlgs": ["RSA256"]
			}
		}
   }`)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID := suite.DecodeOperationID(resp)

	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)

	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "id-ooo",
				GroupsClaim:    "",
				IssuerURL:      "",
				SigningAlgs:    []string{"RSA256"},
				UsernameClaim:  "",
				UsernamePrefix: "",
			},
		},
	})
}

func TestUpdateContext(t *testing.T) {
	suite := NewUpdateSuite(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := procressProvisioning(suite, iid)
	// provisioning done, let's start an update

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       }
   }`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUpdateNotExistingInstance(t *testing.T) {
	suite := NewUpdateSuite(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := procressProvisioning(suite, iid)
	// provisioning done, let's start an update

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/not-existing"),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       }
   }`)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}


func procressProvisioning(suite *UpdateSuite, iid string) *http.Response {
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?plan_id=4deee563-e5ec-4731-b9b1-53b42d855f0c&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		`{
				   "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
				   "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
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
							"clientID": "id-ooo",
							"signingAlgs": ["RSA256"]
						}
			}
   }`)
	operationID := suite.DecodeOperationID(resp)

	// Process provisioning
	suite.WaitForProvisioningState(operationID, domain.InProgress)
	suite.AssertProvisionerStartedProvisioning(operationID)

	suite.FinishProvisioningOperationByProvisioner(operationID)
	// simulate the installed fresh Kyma sets the proper label in the Director
	suite.MarkDirectorWithConsoleURL(operationID)

	// provisioner finishes the operation
	suite.WaitForOperationState(operationID, domain.Succeeded)
	return resp
}
