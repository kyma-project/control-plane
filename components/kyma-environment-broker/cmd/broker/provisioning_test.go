// build provisioning-test
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/google/uuid"
	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
)

const (
	workersAmount int = 5
)

func TestProvisioning_HappyPath(t *testing.T) {
	// given
	suite := NewProvisioningSuite(t, false, "")

	// when
	provisioningOperationID := suite.CreateProvisioning(RuntimeOptions{})

	// then
	suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
	suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

	// when
	suite.FinishProvisioningOperationByProvisionerAndReconciler(provisioningOperationID)

	// then
	suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
	suite.AssertAllStagesFinished(provisioningOperationID)
	suite.AssertProvisioningRequest()
}

func TestProvisioning_Preview(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
					"context": {
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

	suite.WaitForOperationState(opID, domain.Succeeded)

	suite.AssertKymaResourceExists(opID)
	suite.AssertSecretWithKubeconfigExists(opID)
}

func TestProvisioning_AWSWithEURestrictedAccessBadRequest(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu11/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
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
						"region":"us-west-1"
					}
		}`)
	// then
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestProvisioning_AzureWithEURestrictedAccessBadRequest(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-ch20/v2/service_instances/%s?accepts_incomplete=true", iid),
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
						"region":"japaneast"
					}
		}`)
	// then
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestProvisioning_AzureWithEURestrictedAccessHappyFlow(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-ch20/v2/service_instances/%s?accepts_incomplete=true", iid),
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
						"region":"switzerlandnorth"
					}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningAndReconcilingByOperationID(opID)

	// then
	suite.AssertAzureRegion("switzerlandnorth")
}

func TestProvisioning_AzureWithEURestrictedAccessDefaultRegion(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-ch20/v2/service_instances/%s?accepts_incomplete=true", iid),
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
						"name": "testing-cluster"
					}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningAndReconcilingByOperationID(opID)

	// then
	suite.AssertAzureRegion("switzerlandnorth")
}

func TestProvisioning_AWSWithEURestrictedAccessHappyFlow(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu11/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
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
						"region":"eu-central-1"
					}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningAndReconcilingByOperationID(opID)

	// then
	suite.AssertAWSRegionAndZone("eu-central-1")
}

func TestProvisioning_AWSWithEURestrictedAccessDefaultRegion(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu11/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "361c511f-f939-4621-b228-d0fb79a1fe15",
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
	suite.processProvisioningAndReconcilingByOperationID(opID)

	// then
	suite.AssertAWSRegionAndZone("eu-central-1")
}

func TestProvisioning_TrialWithEmptyRegion(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
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
						"region":""
					}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningAndReconcilingByOperationID(opID)

	// then
	suite.AssertAWSRegionAndZone("eu-west-1")
}

func TestProvisioning_Conflict(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
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
						"kymaVersion": "2.4.0"
					}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningAndReconcilingByOperationID(opID)

	// when
	resp = suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
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
						"kymaVersion": "2.5.0"
					}
		}`)
	// then
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestProvisioning_OwnCluster(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "03e3cb66-a4c6-4c6a-b4b0-5d42224debea",
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
						"kubeconfig":"YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmN1cnJlbnQtY29udGV4dDogc2hvb3QtLWt5bWEtZGV2LS1jbHVzdGVyLW5hbWUKY29udGV4dHM6CiAgLSBuYW1lOiBzaG9vdC0ta3ltYS1kZXYtLWNsdXN0ZXItbmFtZQogICAgY29udGV4dDoKICAgICAgY2x1c3Rlcjogc2hvb3QtLWt5bWEtZGV2LS1jbHVzdGVyLW5hbWUKICAgICAgdXNlcjogc2hvb3QtLWt5bWEtZGV2LS1jbHVzdGVyLW5hbWUtdG9rZW4KY2x1c3RlcnM6CiAgLSBuYW1lOiBzaG9vdC0ta3ltYS1kZXYtLWNsdXN0ZXItbmFtZQogICAgY2x1c3RlcjoKICAgICAgc2VydmVyOiBodHRwczovL2FwaS5jbHVzdGVyLW5hbWUua3ltYS1kZXYuc2hvb3QuY2FuYXJ5Lms4cy1oYW5hLm9uZGVtYW5kLmNvbQogICAgICBjZXJ0aWZpY2F0ZS1hdXRob3JpdHktZGF0YTogPi0KICAgICAgICBMUzB0TFMxQ1JVZEpUaUJEUlZKVVNVWkpRMEZVUlMwdExTMHQKdXNlcnM6CiAgLSBuYW1lOiBzaG9vdC0ta3ltYS1kZXYtLWNsdXN0ZXItbmFtZS10b2tlbgogICAgdXNlcjoKICAgICAgdG9rZW46ID4tCiAgICAgICAgdE9rRW4K",
						"shootName": "sh1",
						"shootDomain": "sh1.avs.sap.nothing"
					}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.FinishReconciliation(opID)

	// then
	suite.WaitForOperationState(opID, domain.Succeeded)

	// get instance OSB API call
	// when
	resp = suite.CallAPI("GET", fmt.Sprintf("oauth/v2/service_instances/%s", iid), ``)
	r, e := io.ReadAll(resp.Body)

	// then
	require.NoError(t, e)
	assert.JSONEq(t, `{
  "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
  "plan_id": "03e3cb66-a4c6-4c6a-b4b0-5d42224debea",
  "parameters": {
    "plan_id": "03e3cb66-a4c6-4c6a-b4b0-5d42224debea",
    "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
    "ers_context": {
      "subaccount_id": "sub-id",
      "globalaccount_id": "g-account-id",
      "user_id": "john.smith@email.com"
    },
    "parameters": {
      "name": "testing-cluster",
      "shootName": "sh1",
      "shootDomain": "sh1.avs.sap.nothing"
    },
    "platform_region": "",
    "platform_provider": "unknown"
  },
  "metadata": {
    "labels": {
      "Name": "testing-cluster"
    }
  }
}`, string(r))

}

func TestProvisioning_TrialAtEU(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu11/v2/service_instances/%s?accepts_incomplete=true", iid),
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
	suite.processProvisioningAndReconcilingByOperationID(opID)

	// then
	suite.AssertAWSRegionAndZone("eu-central-1")
}

func TestProvisioning_HandleExistingOperation(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	firstResp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
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

	secondResp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
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

	firstBodyBytes, _ := ioutil.ReadAll(firstResp.Body)
	secondBodyBytes, _ := ioutil.ReadAll(secondResp.Body)

	// then
	assert.Equal(t, string(firstBodyBytes), string(secondBodyBytes))
}

func TestProvisioningWithReconciler_HappyPath(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t, "2.0")
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
			"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
			"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
			"context": {
				"url": "https://sm.url",
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
	suite.processProvisioningAndReconcilingByOperationID(opID)
	provisioningOp, _ := suite.db.Operations().GetProvisioningOperationByID(opID)
	clusterID := provisioningOp.InstanceDetails.ServiceManagerClusterID

	// then
	suite.AssertProvider("aws")
	suite.AssertProvisionRuntimeInputWithoutKymaConfig()

	suite.AssertClusterMetadata(opID, reconcilerApi.Metadata{
		GlobalAccountID: "g-account-id",
		SubAccountID:    "sub-id",
		ServiceID:       "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
		ServicePlanID:   "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
		ServicePlanName: "trial",
		ShootName:       suite.ShootName(opID),
		InstanceID:      iid,
		Region:          "eu-west-1",
	})

	suite.AssertClusterKymaConfig(opID, reconcilerApi.KymaConfig{
		Version:        "2.0",
		Profile:        "Evaluation",
		Administrators: []string{"john.smith@email.com"},
		Components:     suite.fixExpectedComponentListWithSMOperator(opID, clusterID),
	})
	suite.AssertClusterConfigWithKubeconfig(opID)
}

func TestProvisioningWithReconcilerWithBTPOperator_HappyPath(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t, "2.0")
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
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
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"parameters": {
						"name": "testing-cluster"
					}
		}`)

	opID := suite.DecodeOperationID(resp)
	suite.processProvisioningAndReconcilingByOperationID(opID)

	// then
	suite.AssertProvider("aws")
	suite.AssertProvisionRuntimeInputWithoutKymaConfig()

	suite.AssertClusterMetadata(opID, reconcilerApi.Metadata{
		GlobalAccountID: "g-account-id",
		SubAccountID:    "sub-id",
		ServiceID:       "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
		ServicePlanID:   "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
		ServicePlanName: "trial",
		ShootName:       suite.ShootName(opID),
		InstanceID:      iid,
		Region:          "eu-west-1",
	})

	op, _ := suite.db.Operations().GetProvisioningOperationByID(opID)
	suite.AssertClusterKymaConfig(opID, reconcilerApi.KymaConfig{
		Version:        "2.0",
		Profile:        "Evaluation",
		Administrators: []string{"john.smith@email.com"},
		Components:     suite.fixExpectedComponentListWithSMOperator(opID, op.InstanceDetails.ServiceManagerClusterID),
	})

	suite.AssertClusterConfigWithKubeconfig(opID)
}

func TestProvisioning_ClusterParameters(t *testing.T) {
	for tn, tc := range map[string]struct {
		planID                       string
		platformRegion               string
		platformProvider             internal.CloudProvider
		region                       string
		multiZone                    bool
		controlPlaneFailureTolerance string

		expectedZonesCount                  *int
		expectedProfile                     gqlschema.KymaProfile
		expectedProvider                    string
		expectedMinimalNumberOfNodes        int
		expectedMaximumNumberOfNodes        int
		expectedMachineType                 string
		expectedSharedSubscription          bool
		expectedSubscriptionHyperscalerType hyperscaler.Type
	}{
		"Regular trial": {
			planID: broker.TrialPlanID,

			expectedMinimalNumberOfNodes:        1,
			expectedMaximumNumberOfNodes:        1,
			expectedMachineType:                 "Standard_D4_v3",
			expectedProfile:                     gqlschema.KymaProfileEvaluation,
			expectedProvider:                    "azure",
			expectedSharedSubscription:          true,
			expectedSubscriptionHyperscalerType: hyperscaler.Azure,
		},
		"Freemium aws": {
			planID:           broker.FreemiumPlanID,
			platformProvider: internal.AWS,

			expectedMinimalNumberOfNodes:        1,
			expectedMaximumNumberOfNodes:        1,
			expectedProfile:                     gqlschema.KymaProfileEvaluation,
			expectedProvider:                    "aws",
			expectedSharedSubscription:          false,
			expectedMachineType:                 "m5.xlarge",
			expectedSubscriptionHyperscalerType: hyperscaler.AWS,
		},
		"Freemium azure": {
			planID:           broker.FreemiumPlanID,
			platformProvider: internal.Azure,

			expectedMinimalNumberOfNodes:        1,
			expectedMaximumNumberOfNodes:        1,
			expectedProfile:                     gqlschema.KymaProfileEvaluation,
			expectedProvider:                    "azure",
			expectedSharedSubscription:          false,
			expectedMachineType:                 "Standard_D4_v3",
			expectedSubscriptionHyperscalerType: hyperscaler.Azure,
		},
		"Production Azure": {
			planID:    broker.AzurePlanID,
			region:    "westeurope",
			multiZone: false,

			expectedZonesCount:                  ptr.Integer(1),
			expectedMinimalNumberOfNodes:        3,
			expectedMaximumNumberOfNodes:        20,
			expectedMachineType:                 "Standard_D4_v3",
			expectedProfile:                     gqlschema.KymaProfileProduction,
			expectedProvider:                    "azure",
			expectedSharedSubscription:          false,
			expectedSubscriptionHyperscalerType: hyperscaler.Azure,
		},
		"Production Multi-AZ Azure": {
			planID:                       broker.AzurePlanID,
			region:                       "westeurope",
			multiZone:                    true,
			controlPlaneFailureTolerance: "zone",

			expectedZonesCount:                  ptr.Integer(3),
			expectedMinimalNumberOfNodes:        3,
			expectedMaximumNumberOfNodes:        20,
			expectedMachineType:                 "Standard_D4_v3",
			expectedProfile:                     gqlschema.KymaProfileProduction,
			expectedProvider:                    "azure",
			expectedSharedSubscription:          false,
			expectedSubscriptionHyperscalerType: hyperscaler.Azure,
		},
		"Production AWS": {
			planID:    broker.AWSPlanID,
			region:    "us-east-1",
			multiZone: false,

			expectedZonesCount:                  ptr.Integer(1),
			expectedMinimalNumberOfNodes:        3,
			expectedMaximumNumberOfNodes:        20,
			expectedMachineType:                 "m5.xlarge",
			expectedProfile:                     gqlschema.KymaProfileProduction,
			expectedProvider:                    "aws",
			expectedSharedSubscription:          false,
			expectedSubscriptionHyperscalerType: hyperscaler.AWS,
		},
		"Production Multi-AZ AWS": {
			planID:                       broker.AWSPlanID,
			region:                       "us-east-1",
			multiZone:                    true,
			controlPlaneFailureTolerance: "zone",

			expectedZonesCount:                  ptr.Integer(3),
			expectedMinimalNumberOfNodes:        3,
			expectedMaximumNumberOfNodes:        20,
			expectedMachineType:                 "m5.xlarge",
			expectedProfile:                     gqlschema.KymaProfileProduction,
			expectedProvider:                    "aws",
			expectedSharedSubscription:          false,
			expectedSubscriptionHyperscalerType: hyperscaler.AWS,
		},
		"Production GCP": {
			planID:    broker.GCPPlanID,
			region:    "us-central1",
			multiZone: false,

			expectedZonesCount:                  ptr.Integer(1),
			expectedMinimalNumberOfNodes:        3,
			expectedMaximumNumberOfNodes:        20,
			expectedMachineType:                 "n2-standard-4",
			expectedProfile:                     gqlschema.KymaProfileProduction,
			expectedProvider:                    "gcp",
			expectedSharedSubscription:          false,
			expectedSubscriptionHyperscalerType: hyperscaler.GCP,
		},
		"Production Multi-AZ GCP": {
			planID:                       broker.GCPPlanID,
			region:                       "us-central1",
			multiZone:                    true,
			controlPlaneFailureTolerance: "zone",

			expectedZonesCount:                  ptr.Integer(3),
			expectedMinimalNumberOfNodes:        3,
			expectedMaximumNumberOfNodes:        20,
			expectedMachineType:                 "n2-standard-4",
			expectedProfile:                     gqlschema.KymaProfileProduction,
			expectedProvider:                    "gcp",
			expectedSharedSubscription:          false,
			expectedSubscriptionHyperscalerType: hyperscaler.GCP,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			// given
			suite := NewProvisioningSuite(t, tc.multiZone, tc.controlPlaneFailureTolerance)

			// when
			provisioningOperationID := suite.CreateProvisioning(RuntimeOptions{
				PlanID:           tc.planID,
				PlatformRegion:   tc.platformRegion,
				PlatformProvider: tc.platformProvider,
				Region:           tc.region,
			})

			// then
			suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
			suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

			// when
			suite.FinishProvisioningOperationByProvisionerAndReconciler(provisioningOperationID)

			// then
			suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
			suite.AssertAllStagesFinished(provisioningOperationID)

			suite.AssertKymaProfile(provisioningOperationID, tc.expectedProfile)
			suite.AssertProvider(tc.expectedProvider)
			suite.AssertMinimalNumberOfNodes(tc.expectedMinimalNumberOfNodes)
			suite.AssertMaximumNumberOfNodes(tc.expectedMaximumNumberOfNodes)
			suite.AssertMachineType(tc.expectedMachineType)
			suite.AssertZonesCount(tc.expectedZonesCount, tc.planID)
			suite.AssertSubscription(tc.expectedSharedSubscription, tc.expectedSubscriptionHyperscalerType)
			suite.AssertControlPlaneFailureTolerance(tc.controlPlaneFailureTolerance)
		})

	}
}

func TestProvisioning_OIDCValues(t *testing.T) {

	t.Run("should apply default OIDC values when OIDC object is nil", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t, false, "")
		defaultOIDC := fixture.FixOIDCConfigDTO()
		expectedOIDC := gqlschema.OIDCConfigInput{
			ClientID:       defaultOIDC.ClientID,
			GroupsClaim:    defaultOIDC.GroupsClaim,
			IssuerURL:      defaultOIDC.IssuerURL,
			SigningAlgs:    defaultOIDC.SigningAlgs,
			UsernameClaim:  defaultOIDC.UsernameClaim,
			UsernamePrefix: defaultOIDC.UsernamePrefix,
		}

		// when
		provisioningOperationID := suite.CreateProvisioning(RuntimeOptions{})

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
		suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

		// when
		suite.FinishProvisioningOperationByProvisionerAndReconciler(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertOIDC(expectedOIDC)
	})

	t.Run("should apply default OIDC values when all OIDC object's fields are empty", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t, false, "")
		defaultOIDC := fixture.FixOIDCConfigDTO()
		expectedOIDC := gqlschema.OIDCConfigInput{
			ClientID:       defaultOIDC.ClientID,
			GroupsClaim:    defaultOIDC.GroupsClaim,
			IssuerURL:      defaultOIDC.IssuerURL,
			SigningAlgs:    defaultOIDC.SigningAlgs,
			UsernameClaim:  defaultOIDC.UsernameClaim,
			UsernamePrefix: defaultOIDC.UsernamePrefix,
		}
		options := RuntimeOptions{
			OIDC: &internal.OIDCConfigDTO{},
		}

		// when
		provisioningOperationID := suite.CreateProvisioning(options)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
		suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

		// when
		suite.FinishProvisioningOperationByProvisionerAndReconciler(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertOIDC(expectedOIDC)
	})

	t.Run("should apply provided OIDC configuration", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t, false, "")
		providedOIDC := internal.OIDCConfigDTO{
			ClientID:       "fake-client-id-1",
			GroupsClaim:    "fakeGroups",
			IssuerURL:      "https://testurl.local",
			SigningAlgs:    []string{"RS256", "HS256"},
			UsernameClaim:  "fakeUsernameClaim",
			UsernamePrefix: "::",
		}
		expectedOIDC := gqlschema.OIDCConfigInput{
			ClientID:       providedOIDC.ClientID,
			GroupsClaim:    providedOIDC.GroupsClaim,
			IssuerURL:      providedOIDC.IssuerURL,
			SigningAlgs:    providedOIDC.SigningAlgs,
			UsernameClaim:  providedOIDC.UsernameClaim,
			UsernamePrefix: providedOIDC.UsernamePrefix,
		}
		options := RuntimeOptions{OIDC: &providedOIDC}

		// when
		provisioningOperationID := suite.CreateProvisioning(options)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
		suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

		// when
		suite.FinishProvisioningOperationByProvisionerAndReconciler(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertOIDC(expectedOIDC)
	})

	t.Run("should apply default OIDC values on empty OIDC params from input", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t, false, "")
		providedOIDC := internal.OIDCConfigDTO{
			ClientID:  "fake-client-id-1",
			IssuerURL: "https://testurl.local",
		}
		defaultOIDC := defaultOIDCValues()
		expectedOIDC := gqlschema.OIDCConfigInput{
			ClientID:       providedOIDC.ClientID,
			GroupsClaim:    defaultOIDC.GroupsClaim,
			IssuerURL:      providedOIDC.IssuerURL,
			SigningAlgs:    defaultOIDC.SigningAlgs,
			UsernameClaim:  defaultOIDC.UsernameClaim,
			UsernamePrefix: defaultOIDC.UsernamePrefix,
		}
		options := RuntimeOptions{OIDC: &providedOIDC}

		// when
		provisioningOperationID := suite.CreateProvisioning(options)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
		suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

		// when
		suite.FinishProvisioningOperationByProvisionerAndReconciler(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertOIDC(expectedOIDC)
	})
}

func TestProvisioning_RuntimeAdministrators(t *testing.T) {
	t.Run("should use UserID as default value for admins list", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t, false, "")
		options := RuntimeOptions{
			UserID: "fake-user-id",
		}
		expectedAdmins := []string{"fake-user-id"}

		// when
		provisioningOperationID := suite.CreateProvisioning(options)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
		suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

		// when
		suite.FinishProvisioningOperationByProvisionerAndReconciler(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertRuntimeAdmins(expectedAdmins)
	})

	t.Run("should apply new admins list", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t, false, "")
		options := RuntimeOptions{
			UserID:        "fake-user-id",
			RuntimeAdmins: []string{"admin1@test.com", "admin2@test.com"},
		}
		expectedAdmins := []string{"admin1@test.com", "admin2@test.com"}

		// when
		provisioningOperationID := suite.CreateProvisioning(options)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
		suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

		// when
		suite.FinishProvisioningOperationByProvisionerAndReconciler(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertRuntimeAdmins(expectedAdmins)
	})

	t.Run("should apply empty admin value (list is not empty)", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t, false, "")
		options := RuntimeOptions{
			UserID:        "fake-user-id",
			RuntimeAdmins: []string{""},
		}
		expectedAdmins := []string{""}

		// when
		provisioningOperationID := suite.CreateProvisioning(options)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
		suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

		// when
		suite.FinishProvisioningOperationByProvisionerAndReconciler(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertRuntimeAdmins(expectedAdmins)
	})
}

func TestProvisioning_WithoutNetworkFilter(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t, "2.0")
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
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
	suite.processProvisioningAndReconcilingByOperationID(opID)
	instance := suite.GetInstance(iid)

	// then
	disabled := false
	suite.AssertDisabledNetworkFilterForProvisioning(&disabled)
	assert.Nil(suite.t, instance.Parameters.ErsContext.LicenseType)
}

func TestProvisioning_WithNetworkFilter(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t, "2.0")
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
					"context": {
						"sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
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
	suite.processProvisioningAndReconcilingByOperationID(opID)
	instance := suite.GetInstance(iid)

	// then
	disabled := true
	suite.AssertDisabledNetworkFilterForProvisioning(&disabled)
	assert.Equal(suite.t, "CUSTOMER", *instance.Parameters.ErsContext.LicenseType)
}

func TestProvisioning_PRVersionWithoutOverrides(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-ch20/v2/service_instances/%s?accepts_incomplete=true", iid),
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
						"overridesVersion":"",
						"kymaVersion":"PR-99999"				
					}
		}`)
	opID := suite.DecodeOperationID(resp)

	// then
	suite.WaitForProvisioningState(opID, domain.Failed)
}
