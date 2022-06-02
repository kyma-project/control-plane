package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdate(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
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
							"clientID": "id-initial",
							"signingAlgs": ["PS512"],
                            "issuerURL": "https://issuer.url.com"
						}
			}
   }`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	// when
	// OSB update:
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
		"parameters": {
			"oidc": {
				"clientID": "id-ooo",
				"signingAlgs": ["RS256"],
                "issuerURL": "https://issuer.url.com"
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
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url.com",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
		},
		Administrators: []string{"john.smith@email.com"},
	})
}

func TestUpdateFailedInstance(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
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
						"name": "testing-cluster"
				}
   }`)
	opID := suite.DecodeOperationID(resp)
	suite.failProvisioningByOperationID(opID)

	// when
	// OSB update:
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
		"parameters": {
			"oidc": {
				"clientID": "id-ooo",
				"signingAlgs": ["RSA256"],
                "issuerURL": "https://issuer.url.com"
			}
		}
   }`)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	errResponse := suite.DecodeErrorResponse(resp)

	assert.Equal(t, "Unable to process an update of a failed instance", errResponse.Description)
}

func TestUpdateDeprovisioningInstance(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
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
						"name": "testing-cluster"
				}
   }`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	// deprovision
	resp = suite.CallAPI("DELETE", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
		``)
	depOpID := suite.DecodeOperationID(resp)

	suite.WaitForOperationState(depOpID, domain.InProgress)

	// when
	// OSB update:
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
		"parameters": {
			"oidc": {
				"clientID": "id-ooo",
				"signingAlgs": ["RSA256"],
                "issuerURL": "https://issuer.url.com"
			}
		}
   }`)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	errResponse := suite.DecodeErrorResponse(resp)

	assert.Equal(t, "Unable to process an update of a deprovisioned instance", errResponse.Description)
}

func TestUpdateWithNoOIDCParams(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
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
						"name": "testing-cluster"
				}
   }`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	// when
	// OSB update:
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
		"parameters": {
		}
   }`)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID := suite.DecodeOperationID(resp)

	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)

	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: defaultOIDCConfig(),
		},
		Administrators: []string{"john.smith@email.com"},
	})
}

func TestUpdateWithNoOidcOnUpdate(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
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
							"clientID": "id-ooo",
							"signingAlgs": ["RS256"],
                            "issuerURL": "https://issuer.url.com"
						}
			}
   }`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	// when
	// OSB update:
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
		"parameters": {
			
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
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url.com",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
		},
		Administrators: []string{"john.smith@email.com"},
	})
}

func TestUpdateContext(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
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
							"clientID": "id-ooo",
							"signingAlgs": ["RS384"],
                            "issuerURL": "https://issuer.url.com"
						}
			}
   }`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	// when
	// OSB update
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       }
   }`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUnsuspensionTrialKyma20(t *testing.T) {
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
                         "kymaVersion":"2.0"
			}
   }`)
	opID := suite.DecodeOperationID(resp)
	suite.processReconcilingByOperationID(opID)

	suite.Log("*** Suspension ***")

	// Process Suspension
	// OSB context update (suspension)
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com",
           "active": false
       }
   }`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	suspensionOpID := suite.WaitForLastOperation(iid, domain.InProgress)

	suite.MarkClustertConfigurationDeleted(iid)
	suite.FinishDeprovisioningOperationByProvisioner(suspensionOpID)
	suite.WaitForOperationState(suspensionOpID, domain.Succeeded)
	suite.RemoveFromReconcilerByInstanceID(iid)

	// OSB update
	suite.Log("*** Unsuspension ***")
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com",
			"active": true
       }
       
   }`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	suite.processReconciliationByInstanceID(iid)

}

func TestUnsuspensionTrialWithDefaultProviderChangedForNonDefaultRegion(t *testing.T) {
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-us10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", iid),
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
						"name": "testing-cluster"
			}
   }`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	suite.Log("*** Suspension ***")

	// Process Suspension
	// OSB context update (suspension)
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-us10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com",
           "active": false
       }
   }`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	suspensionOpID := suite.WaitForLastOperation(iid, domain.InProgress)

	suite.FinishDeprovisioningOperationByProvisioner(suspensionOpID)
	suite.WaitForOperationState(suspensionOpID, domain.Succeeded)

	// WHEN
	suite.ChangeDefaultTrialProvider(internal.AWS)
	// OSB update
	suite.Log("*** Unsuspension ***")
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-us10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com",
			"active": true
       }
       
   }`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	suite.processProvisioningByInstanceID(iid)

	// check that the region and zone is set
	suite.AssertAWSRegionAndZone("us-east-1")
}

func TestUpdateOidcForSuspendedInstance(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	// uncomment to see graphql queries
	//suite.EnableDumpingProvisionerRequests()
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
							"clientID": "id-ooo",
							"signingAlgs": ["RS256"],
                            "issuerURL": "https://issuer.url.com"
						}
			}
   }`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	suite.Log("*** Suspension ***")

	// Process Suspension
	// OSB context update (suspension)
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com",
           "active": false
       }
   }`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	suspensionOpID := suite.WaitForLastOperation(iid, domain.InProgress)

	suite.FinishDeprovisioningOperationByProvisioner(suspensionOpID)
	suite.WaitForOperationState(suspensionOpID, domain.Succeeded)

	// WHEN
	// OSB update
	suite.Log("*** Update ***")
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
       "parameters": {
       		"oidc": {
				"clientID": "id-oooxx",
				"signingAlgs": ["RS256"],
                "issuerURL": "https://issuer.url.com"
			}
       }
   }`)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOpID := suite.DecodeOperationID(resp)
	suite.WaitForOperationState(updateOpID, domain.Succeeded)

	// THEN
	instance := suite.GetInstance(iid)
	assert.Equal(t, "id-oooxx", instance.Parameters.Parameters.OIDC.ClientID)

	// Start unsuspension
	// OSB update (unsuspension)
	suite.Log("*** Update (unsuspension) ***")
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com",
           "active": true
       }
   }`)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// WHEN
	suite.processProvisioningByInstanceID(iid)

	// THEN
	instance = suite.GetInstance(iid)
	assert.Equal(t, "id-oooxx", instance.Parameters.Parameters.OIDC.ClientID)
	input := suite.LastProvisionInput(iid)
	assert.Equal(t, "id-oooxx", input.ClusterConfig.GardenerConfig.OidcConfig.ClientID)
}

func TestUpdateNotExistingInstance(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
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
							"clientID": "id-ooo",
							"signingAlgs": ["RS256"],
                            "issuerURL": "https://issuer.url.com"
						}
			}
   }`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)
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

func TestUpdateDefaultAdminNotChanged(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()
	expectedAdmins := []string{"john.smith@email.com"}

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
						"name": "testing-cluster"
			}
   }`)

	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
			"user_id": "jack.anvil@email.com"
       },
		"parameters": {
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
		},
		Administrators: expectedAdmins,
	})
}

func TestUpdateDefaultAdminNotChangedWithCustomOIDC(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()
	expectedAdmins := []string{"john.smith@email.com"}

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
							"clientID": "id-ooo",
                            "issuerURL": "https://issuer.url.com"
						}
			}
   }`)

	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
			"user_id": "jack.anvil@email.com"
       },
		"parameters": {
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "id-ooo",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url.com",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
		},
		Administrators: expectedAdmins,
	})
}

func TestUpdateDefaultAdminNotChangedWithOIDCUpdate(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()
	expectedAdmins := []string{"john.smith@email.com"}

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
						"name": "testing-cluster"
			}
   }`)

	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
			"user_id": "jack.anvil@email.com"
       },
		"parameters": {
			"oidc": {
				"clientID": "id-ooo",
				"signingAlgs": ["RS384"],
				"issuerURL": "https://issuer.url.com",
				"groupsClaim": "new-groups-claim",
				"usernameClaim": "new-username-claim",
				"usernamePrefix": "->"
			}
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "id-ooo",
				GroupsClaim:    "new-groups-claim",
				IssuerURL:      "https://issuer.url.com",
				SigningAlgs:    []string{"RS384"},
				UsernameClaim:  "new-username-claim",
				UsernamePrefix: "->",
			},
		},
		Administrators: expectedAdmins,
	})
}

func TestUpdateDefaultAdminOverwritten(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()
	expectedAdmins := []string{"newAdmin1@kyma.cx", "newAdmin2@kyma.cx"}

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
						"name": "testing-cluster"
			}
   }`)

	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
			"user_id": "jack.anvil@email.com"
       },
		"parameters": {
			"administrators":["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"]
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
		},
		Administrators: expectedAdmins,
	})
	suite.AssertInstanceRuntimeAdmins(id, expectedAdmins)
}

func TestUpdateCustomAdminsNotChanged(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()
	expectedAdmins := []string{"newAdmin1@kyma.cx", "newAdmin2@kyma.cx"}

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
						"administrators":["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"]
			}
   }`)

	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "jack.anvil@email.com"
       },
		"parameters": {
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
		},
		Administrators: expectedAdmins,
	})
	suite.AssertInstanceRuntimeAdmins(id, expectedAdmins)
}

func TestUpdateCustomAdminsNotChangedWithOIDCUpdate(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()
	expectedAdmins := []string{"newAdmin1@kyma.cx", "newAdmin2@kyma.cx"}

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
						"administrators":["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"]
			}
   }`)

	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id"
       },
		"parameters": {
			"oidc": {
				"clientID": "id-ooo",
				"signingAlgs": ["ES256"],
				"issuerURL": "https://newissuer.url.com"
			}
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "id-ooo",
				GroupsClaim:    "groups",
				IssuerURL:      "https://newissuer.url.com",
				SigningAlgs:    []string{"ES256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
		},
		Administrators: expectedAdmins,
	})
	suite.AssertInstanceRuntimeAdmins(id, expectedAdmins)
}

func TestUpdateCustomAdminsOverwritten(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()
	expectedAdmins := []string{"newAdmin3@kyma.cx", "newAdmin4@kyma.cx"}

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
						"administrators":["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"]
			}
   }`)

	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "jack.anvil@email.com"
       },
		"parameters": {
			"administrators":["newAdmin3@kyma.cx", "newAdmin4@kyma.cx"]
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
		},
		Administrators: expectedAdmins,
	})
	suite.AssertInstanceRuntimeAdmins(id, expectedAdmins)
}

func TestUpdateCustomAdminsOverwrittenWithOIDCUpdate(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()
	expectedAdmins := []string{"newAdmin3@kyma.cx", "newAdmin4@kyma.cx"}

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
						"administrators":["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"]
			}
   }`)

	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com"
       },
		"parameters": {
			"oidc": {
				"clientID": "id-ooo",
				"signingAlgs": ["ES384"],
				"issuerURL": "https://issuer.url.com",
				"groupsClaim": "new-groups-claim"
			},
			"administrators":["newAdmin3@kyma.cx", "newAdmin4@kyma.cx"]
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "id-ooo",
				GroupsClaim:    "new-groups-claim",
				IssuerURL:      "https://issuer.url.com",
				SigningAlgs:    []string{"ES384"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
		},
		Administrators: expectedAdmins,
	})
	suite.AssertInstanceRuntimeAdmins(id, expectedAdmins)
}

func TestUpdateCustomAdminsOverwrittenTwice(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()
	expectedAdmins1 := []string{"newAdmin3@kyma.cx", "newAdmin4@kyma.cx"}
	expectedAdmins2 := []string{"newAdmin5@kyma.cx", "newAdmin6@kyma.cx"}

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id),
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
						"administrators":["newAdmin1@kyma.cx", "newAdmin2@kyma.cx"]
			}
   }`)

	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "jack.anvil@email.com"
       },
		"parameters": {
			"administrators":["newAdmin3@kyma.cx", "newAdmin4@kyma.cx"]
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
		},
		Administrators: expectedAdmins1,
	})
	suite.AssertInstanceRuntimeAdmins(id, expectedAdmins1)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id"
       },
		"parameters": {
			"oidc": {
				"clientID": "id-ooo",
				"signingAlgs": ["PS256"],
				"issuerURL": "https://newissuer.url.com",
				"usernamePrefix": "->"
			},
			"administrators":["newAdmin5@kyma.cx", "newAdmin6@kyma.cx"]
		}
   }`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID = suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "id-ooo",
				GroupsClaim:    "groups",
				IssuerURL:      "https://newissuer.url.com",
				SigningAlgs:    []string{"PS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "->",
			},
		},
		Administrators: expectedAdmins2,
	})
	suite.AssertInstanceRuntimeAdmins(id, expectedAdmins2)
}

func TestUpdateAutoscalerParams(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id), `
{
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
		"autoScalerMin":5,
		"autoScalerMax":7,
		"maxSurge":3,
		"maxUnavailable":4
	}
}`)

	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "jack.anvil@email.com"
	},
	"parameters": {
		"autoScalerMin":15,
		"autoScalerMax":25,
		"maxSurge":10,
		"maxUnavailable":7
	}
}`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// when
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	min, max, surge, unav := 15, 25, 10, 7
	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			AutoScalerMin:  &min,
			AutoScalerMax:  &max,
			MaxSurge:       &surge,
			MaxUnavailable: &unav,
		},
		Administrators: []string{"john.smith@email.com"},
	})
}

func TestUpdateAutoscalerWrongParams(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id), `
{
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
		"autoScalerMin":5,
		"autoScalerMax":7,
		"maxSurge":3,
		"maxUnavailable":4
	}
}`)

	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "jack.anvil@email.com"
	},
	"parameters": {
		"autoScalerMin":26,
		"autoScalerMax":25,
		"maxSurge":10,
		"maxUnavailable":7
	}
}`)

	// then
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestUpdateAutoscalerPartialSequence(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id), `
{
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
		"name": "testing-cluster"
	}
}`)

	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "jack.anvil@email.com"
	},
	"parameters": {
		"autoScalerMin":15
	}
}`)

	// then
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "jack.anvil@email.com"
	},
	"parameters": {
		"autoScalerMax":15
	}
}`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)
	max := 15
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			AutoScalerMax: &max,
		},
		Administrators: []string{"john.smith@email.com"},
	})

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "jack.anvil@email.com"
	},
	"parameters": {
		"autoScalerMin":14
	}
}`)

	// then
	suite.WaitForOperationState(upgradeOperationID, domain.Succeeded)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID = suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)
	min := 14
	suite.AssertShootUpgrade(upgradeOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			AutoScalerMin: &min,
		},
		Administrators: []string{"john.smith@email.com"},
	})

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "jack.anvil@email.com"
	},
	"parameters": {
		"autoScalerMin":16
	}
}`)

	// then
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestUpdateWhenBothErsContextAndUpdateParametersProvided(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	// uncomment to see graphql queries
	//suite.EnableDumpingProvisionerRequests()
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
							"clientID": "id-ooo",
							"signingAlgs": ["RS256"],
                            "issuerURL": "https://issuer.url.com"
						}
			}
   }`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningByOperationID(opID)

	suite.Log("*** Suspension ***")

	// when
	// Process Suspension
	// OSB context update (suspension)
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
       "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
       "plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
       "context": {
           "globalaccount_id": "g-account-id",
           "user_id": "john.smith@email.com",
           "active": false
       },
       "parameters": {
			"name": "testing-cluster"
		}
   }`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	suspensionOpID := suite.WaitForLastOperation(iid, domain.InProgress)

	suite.FinishDeprovisioningOperationByProvisioner(suspensionOpID)
	suite.WaitForOperationState(suspensionOpID, domain.Succeeded)

	// THEN
	lastOp, err := suite.db.Operations().GetLastOperation(iid)
	require.NoError(t, err)
	assert.Equal(t, internal.OperationTypeDeprovision, lastOp.Type, "last operation should be type deprovision")

	updateOps, err := suite.db.Operations().ListUpdatingOperationsByInstanceID(iid)
	require.NoError(t, err)
	assert.Len(t, updateOps, 0, "should not create any update operations")
}

func TestUpdateSCMigrationSuccess(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	mockBTPOperatorClusterID()
	defer suite.TearDown()
	id := "InstanceID-SCMigration"

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true&plan_id=7d55d31d-35ae-4438-bf13-6ffdfa107d9f&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
	"context": {
		"sm_platform_credentials": {
			"url": "https://sm.url",
			"credentials": {
				"basic": {
					"username": "u-name",
					"password": "pass"
				}
			}
		},
		"globalaccount_id": "g-account-id",
		"subaccount_id": "sub-id",
		"user_id": "john.smith@email.com"
	},
	"parameters": {
		"name": "testing-cluster",
		"kymaVersion": "2.0"
	}
}`)

	opID := suite.DecodeOperationID(resp)
	suite.processReconcilingByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)
	i, err := suite.db.Instances().GetByID(id)
	assert.NoError(t, err, "getting instance after provisioning, before update")
	rs, err := suite.db.RuntimeStates().GetLatestWithReconcilerInputByRuntimeID(i.RuntimeID)
	if rs.ClusterSetup == nil {
		t.Fatal("expected cluster setup post provisioning kyma 2.0 cluster")
	}
	if rs.ClusterSetup.KymaConfig.Version != "2.0" {
		t.Fatalf("expected cluster setup kyma config version to match 2.0, got %v", rs.ClusterSetup.KymaConfig.Version)
	}
	assert.Equal(t, opID, rs.OperationID, "runtime state provisioning operation ID")
	assert.NoError(t, err, "getting runtime state after provisioning, before update")
	assert.ElementsMatch(t, rs.KymaConfig.Components, []*gqlschema.ComponentConfigurationInput{})
	assert.ElementsMatch(t, componentNames(rs.ClusterSetup.KymaConfig.Components), []string{"service-catalog-addons", "ory", "monitoring", "helm-broker", "service-manager-proxy", "service-catalog"})

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "john.smith@email.com",
		"sm_operator_credentials": {
			"clientid": "testClientID",
			"clientsecret": "testClientSecret",
			"sm_url": "https://service-manager.kyma.com",
			"url": "https://test.auth.com",
			"xsappname": "testXsappname"
		},
		"isMigration": true
	}
}`)

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)
	time.Sleep(5 * time.Millisecond)
	suite.FinishUpdatingOperationByReconciler(updateOperationID)

	// check first call to reconciler installing BTP-Operator and sc-migration, disabling SVCAT
	rsu1, err := suite.db.RuntimeStates().GetLatestWithReconcilerInputByRuntimeID(i.RuntimeID)
	assert.NoError(t, err, "getting runtime mid update")
	assert.Equal(t, updateOperationID, rsu1.OperationID, "runtime state update operation ID")
	assert.ElementsMatch(t, rsu1.KymaConfig.Components, []*gqlschema.ComponentConfigurationInput{})
	assert.ElementsMatch(t, componentNames(rs.ClusterSetup.KymaConfig.Components), []string{"ory", "monitoring", "btp-operator", "sc-migration"})

	// check second call to reconciler and see that sc-migration is no longer present and svcat related components are gone as well
	time.Sleep(5 * time.Millisecond)
	suite.FinishUpdatingOperationByReconciler(updateOperationID)

	i, err = suite.db.Instances().GetByID(id)
	assert.NoError(t, err, "getting instance after update")
	assert.True(t, i.InstanceDetails.SCMigrationTriggered, "instance SCMigrationTriggered after update")
	rsu2, err := suite.db.RuntimeStates().GetLatestWithReconcilerInputByRuntimeID(i.RuntimeID)
	assert.NoError(t, err, "getting runtime after update")
	assert.NotEqual(t, rsu1.ID, rsu2.ID, "runtime_state ID from first call should differ runtime_state ID from second call")
	assert.Equal(t, updateOperationID, rsu2.OperationID, "runtime state update operation ID")
	assert.ElementsMatch(t, rsu2.KymaConfig.Components, []*gqlschema.ComponentConfigurationInput{})
	assert.ElementsMatch(t, componentNames(rsu2.ClusterSetup.KymaConfig.Components), []string{"ory", "monitoring", "btp-operator"})
	for _, c := range rsu2.ClusterSetup.KymaConfig.Components {
		if c.Component == "btp-operator" {
			exp := reconcilerApi.Component{
				Component: "btp-operator",
				Namespace: "kyma-system",
				URL:       "https://btp-operator",
				Configuration: []reconcilerApi.Configuration{
					{Key: "manager.secret.clientid", Value: "testClientID", Secret: true},
					{Key: "manager.secret.clientsecret", Value: "testClientSecret", Secret: true},
					{Key: "manager.secret.url", Value: "https://service-manager.kyma.com"},
					{Key: "manager.secret.sm_url", Value: "https://service-manager.kyma.com"},
					{Key: "manager.secret.tokenurl", Value: "https://test.auth.com"},
					{Key: "cluster.id", Value: "cluster_id"},
				},
			}
			assert.Equal(t, exp, c)
		}
	}

	// finalize second call to reconciler and wait for the operation to finish
	//suite.AssertReconcilerStartedReconcilingWhenUpgrading(instanceID)
	time.Sleep(5 * time.Millisecond)
	suite.FinishUpdatingOperationByReconciler(updateOperationID)
	suite.WaitForOperationState(updateOperationID, domain.Succeeded)

	// change component input (additional components) and see if it works with update operation
	suite.componentProvider.decorator["btp-operator"] = v1alpha1.KymaComponent{
		Name:      "btp-operator",
		Namespace: "kyma-system",
		Source:    &v1alpha1.ComponentSource{URL: "https://btp-operator/updated"},
	}
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "john.smith@email.com",
		"sm_operator_credentials": {
			"clientid": "testClientID",
			"clientsecret": "testClientSecret",
			"sm_url": "https://service-manager.kyma.com",
			"url": "https://test.auth.com",
			"xsappname": "testXsappname"
		},
		"isMigration": true
	}
}`)

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	update2OperationID := suite.DecodeOperationID(resp)
	time.Sleep(5 * time.Millisecond)
	suite.FinishUpdatingOperationByReconciler(update2OperationID)
	i, err = suite.db.Instances().GetByID(id)
	assert.NoError(t, err, "getting instance after second update")
	assert.True(t, i.InstanceDetails.SCMigrationTriggered, "instance SCMigrationTriggered after second update")
	rsu3, err := suite.db.RuntimeStates().GetLatestWithReconcilerInputByRuntimeID(i.RuntimeID)
	assert.NoError(t, err, "getting runtime after second update")
	assert.NotEqual(t, rsu2.ID, rsu3.ID, "runtime_state ID from second call should differ runtime_state ID from third call")
	assert.Equal(t, update2OperationID, rsu3.OperationID, "runtime state second update operation ID")
	assert.ElementsMatch(t, rsu3.KymaConfig.Components, []*gqlschema.ComponentConfigurationInput{})
	assert.ElementsMatch(t, componentNames(rsu3.ClusterSetup.KymaConfig.Components), []string{"ory", "monitoring", "btp-operator", "sc-migration"})
	for _, c := range rsu3.ClusterSetup.KymaConfig.Components {
		if c.Component == "btp-operator" {
			exp := reconcilerApi.Component{
				Component: "btp-operator",
				Namespace: "kyma-system",
				URL:       "https://btp-operator/updated",
				Configuration: []reconcilerApi.Configuration{
					{Key: "manager.secret.clientid", Value: "testClientID", Secret: true},
					{Key: "manager.secret.clientsecret", Value: "testClientSecret", Secret: true},
					{Key: "manager.secret.url", Value: "https://service-manager.kyma.com"},
					{Key: "manager.secret.sm_url", Value: "https://service-manager.kyma.com"},
					{Key: "manager.secret.tokenurl", Value: "https://test.auth.com"},
					{Key: "cluster.id", Value: "cluster_id"},
				},
			}
			assert.Equal(t, exp, c)
		}
	}

}

func TestUpdateNetworkFilterPersisted(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t, "2.0")
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
						"sm_operator_credentials": {
							"clientid": "testClientID",
							"clientsecret": "testClientSecret",
							"sm_url": "https://service-manager.kyma.com",
							"url": "https://test.auth.com",
							"xsappname": "testXsappname"
						},
						"license_type": "CUSTOMER",
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
	instance := suite.GetInstance(id)

	// then
	disabled := true
	suite.AssertDisabledNetworkFilterForProvisioning(&disabled)
	assert.Equal(suite.t, "CUSTOMER", *instance.Parameters.ErsContext.LicenseType)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
		{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
			"context": {
				"globalaccount_id": "g-account-id",
				"user_id": "john.smith@email.com",
				"sm_operator_credentials": {
					"clientid": "testClientID",
					"clientsecret": "testClientSecret",
					"sm_url": "https://service-manager.kyma.com",
					"url": "https://test.auth.com",
					"xsappname": "testXsappname"
				}
			},
			"parameters": {
				"name": "testing-cluster"
			}
		}`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisionerAndReconciler(updateOperationID)
	suite.WaitForOperationState(updateOperationID, domain.Succeeded)
	updateOp, _ := suite.db.Operations().GetUpdatingOperationByID(updateOperationID)
	assert.NotNil(suite.t, updateOp.ProvisioningParameters.ErsContext.LicenseType)
	suite.AssertDisabledNetworkFilterRuntimeState(instance.RuntimeID, updateOperationID, &disabled)
	instance2 := suite.GetInstance(id)
	assert.Equal(suite.t, "CUSTOMER", *instance2.Parameters.ErsContext.LicenseType)
}

func TestUpdateStoreNetworkFilterWhileSVCATMigration(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t, "2.0")
	mockBTPOperatorClusterID()
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
			"context": {
				"sm_platform_credentials": {
					"url": "https://sm.url",
					"credentials": {
						"basic": {
							"username": "u-name",
							"password": "pass"
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
	instance := suite.GetInstance(id)

	// then
	suite.AssertDisabledNetworkFilterForProvisioning(nil)
	suite.AssertDisabledNetworkFilterRuntimeState(instance.RuntimeID, opID, nil)
	assert.Nil(suite.t, instance.Parameters.ErsContext.LicenseType)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
		{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
			"context": {
				"globalaccount_id": "g-account-id",
				"user_id": "john.smith@email.com",
				"sm_operator_credentials": {
					"clientid": "testClientID",
					"clientsecret": "testClientSecret",
					"sm_url": "https://service-manager.kyma.com",
					"url": "https://test.auth.com",
					"xsappname": "testXsappname2"
				},
				"license_type": "CUSTOMER",
				"isMigration": true
			}
		}`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByReconcilerBoth(updateOperationID)
	suite.WaitForOperationState(updateOperationID, domain.Succeeded)
	instance2 := suite.GetInstance(id)
	// license_type should be stored in the instance table for ERS context and future upgrades
	// but shouldn't be sent to provisioner when migration is triggered
	suite.AssertDisabledNetworkFilterForProvisioning(nil)
	assert.Equal(suite.t, "CUSTOMER", *instance2.Parameters.ErsContext.LicenseType)

	// when
	// second update without triggering migration
	// it should be fine if ERS omits license_type and KEB should reuse the last applied value
	// because migration wasn't triggered, KEB should send payload to provisioner with network filter disabled
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
		{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
			"context": {
				"globalaccount_id": "g-account-id",
				"user_id": "john.smith@email.com",
				"sm_operator_credentials": {
					"clientid": "testClientID",
					"clientsecret": "testClientSecret",
					"sm_url": "https://service-manager.kyma.com",
					"url": "https://test.auth.com",
					"xsappname": "testXsappname2"
				}
			}
		}`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperation2ID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(updateOperation2ID)
	suite.WaitForOperationState(updateOperation2ID, domain.Succeeded)
	instance3 := suite.GetInstance(id)
	assert.Equal(suite.t, "CUSTOMER", *instance3.Parameters.ErsContext.LicenseType)
	disabled := true
	suite.AssertDisabledNetworkFilterRuntimeState(instance.RuntimeID, updateOperation2ID, &disabled)
}

func TestUpdateStoreNetworkFilterAndUpdate(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t, "2.0")
	mockBTPOperatorClusterID()
	defer suite.TearDown()
	id := uuid.New().String()

	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", id),
		`{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
			"context": {
				"sm_operator_credentials": {
					"clientid": "testClientID",
					"clientsecret": "testClientSecret",
					"sm_url": "https://service-manager.kyma.com",
					"url": "https://test.auth.com",
					"xsappname": "testXsappname2"
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
	instance := suite.GetInstance(id)

	// then
	suite.AssertDisabledNetworkFilterForProvisioning(nil)
	assert.Nil(suite.t, instance.Parameters.ErsContext.LicenseType)

	// when
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", id), `
		{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
			"context": {
				"globalaccount_id": "g-account-id",
				"user_id": "john.smith@email.com",
				"sm_operator_credentials": {
					"clientid": "testClientID",
					"clientsecret": "testClientSecret",
					"sm_url": "https://service-manager.kyma.com",
					"url": "https://test.auth.com",
					"xsappname": "testXsappname"
				},
				"license_type": "CUSTOMER"
			}
		}`)

	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)
	updateOp, _ := suite.db.Operations().GetUpdatingOperationByID(updateOperationID)
	assert.NotNil(suite.t, updateOp.ProvisioningParameters.ErsContext.LicenseType)
	instance2 := suite.GetInstance(id)
	// license_type should be stored in the instance table for ERS context and future upgrades
	// as well as sent to provisioner because the migration has not been triggered
	disabled := true
	suite.AssertDisabledNetworkFilterRuntimeState(instance.RuntimeID, updateOperationID, &disabled)
	assert.Equal(suite.t, "CUSTOMER", *instance2.Parameters.ErsContext.LicenseType)
	suite.FinishUpdatingOperationByProvisionerAndReconciler(updateOperationID)
	suite.WaitForOperationState(updateOperationID, domain.Succeeded)
}

func componentNames(components []reconcilerApi.Component) []string {
	names := make([]string, 0, len(components))
	for _, c := range components {
		names = append(names, c.Component)
	}
	return names
}
