package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestClusterUpgradeUsesUpdatedAutoscalerParams(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// Create an SKR with a default autoscaler params
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
					"context": {
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com",
                        "sm_platform_credentials": {
							  "url": "https://sm.url",
							  "credentials": {}
					    }
					},
					"parameters": {
						"name": "testing-cluster",
						"kymaVersion": "2.0"
					}
		}`)
	opID := suite.DecodeOperationID(resp)
	suite.processReconcilingByOperationID(opID)
	suite.WaitForOperationState(opID, domain.Succeeded)

	// perform an update with custom autoscaler params
	resp = suite.CallAPI("PATCH", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid), `
{
	"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
	"plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
	"context": {
		"globalaccount_id": "g-account-id",
		"user_id": "jack.anvil@email.com"
	},
	"parameters": {
		"autoScalerMin":150,
		"autoScalerMax":250,
		"maxSurge":13,
		"maxUnavailable":9
	}
}`)
	// then
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	upgradeOperationID := suite.DecodeOperationID(resp)
	suite.FinishUpdatingOperationByProvisioner(upgradeOperationID)

	// when
	orchestrationResp := suite.CallAPI("POST", "upgrade/cluster",
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
				}
				}`)
	oID := suite.DecodeOrchestrationID(orchestrationResp)

	var upgradeKymaOperationID string
	err := wait.PollImmediate(5*time.Millisecond, 400*time.Millisecond, func() (bool, error) {
		var err error
		opResponse := suite.CallAPI("GET", fmt.Sprintf("orchestrations/%s/operations", oID), "")
		upgradeKymaOperationID, err = suite.DecodeLastUpgradeKymaOperationIDFromOrchestration(opResponse)
		return err == nil, nil
	})

	require.NoError(t, err)

	// then
	disabled := false
	suite.AssertShootUpgrade(upgradeKymaOperationID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion:   ptr.String("1.18"),
			MachineType:         ptr.String("Standard_D4_v3"),
			MachineImage:        ptr.String("coreos"),
			MachineImageVersion: ptr.String("253"),

			AutoScalerMin:  ptr.Integer(150),
			AutoScalerMax:  ptr.Integer(250),
			MaxSurge:       ptr.Integer(13),
			MaxUnavailable: ptr.Integer(9),

			EnableKubernetesVersionAutoUpdate:   ptr.Bool(false),
			EnableMachineImageVersionAutoUpdate: ptr.Bool(false),

			OidcConfig:                    defaultOIDCConfig(),
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: []string{"john.smith@email.com"},
	})

}

func TestKymaUpgrade_UpgradeAfterMigration(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t, "2.0.0-rc4", "2.0.0")
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
		"name": "testing-cluster",
		"kymaVersion": "2.0.0-rc4"
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
		"isMigration": true
	}
}`)

	// make sure migration finished
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	updateOperationID := suite.DecodeOperationID(resp)
	time.Sleep(5 * time.Millisecond)
	suite.FinishUpdatingOperationByReconciler(updateOperationID)
	time.Sleep(5 * time.Millisecond)
	suite.FinishUpdatingOperationByReconciler(updateOperationID)
	suite.WaitForOperationState(updateOperationID, domain.Succeeded)

	// ensure component list after update is correct
	i, err := suite.db.Instances().GetByID(id)
	assert.NoError(t, err, "getting instance after update")
	rsu1, err := suite.db.RuntimeStates().GetLatestWithReconcilerInputByRuntimeID(i.RuntimeID)
	assert.NoError(t, err, "getting runtime after update")
	assert.True(t, i.InstanceDetails.SCMigrationTriggered, "instance SCMigrationTriggered after update")
	assert.Equal(t, updateOperationID, rsu1.OperationID, "runtime state update operation ID")
	assert.ElementsMatch(t, componentNames(rsu1.ClusterSetup.KymaConfig.Components), []string{"ory", "monitoring", "btp-operator"})

	// run upgrade
	orchestrationResp := suite.CallAPI("POST", "upgrade/kyma", `
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
	"kyma": {
		"version": "2.0.0"
	}
}`)
	oID := suite.DecodeOrchestrationID(orchestrationResp)

	// wait for orchestration to be processed and operations created for that orchestration
	wait.Poll(time.Millisecond*10, time.Second, func() (bool, error) {
		op, err := suite.db.Operations().GetLastOperation(instanceID)
		if err != nil {
			return false, nil // poll again
		}
		if op.Type == "upgradeKyma" {
			return true, nil // it got created
		}
		return false, nil // last operation != kyma upgrade, poll again
	})

	opResponse := suite.CallAPI("GET", fmt.Sprintf("orchestrations/%s/operations", oID), "")
	upgradeKymaOperationID, err := suite.DecodeLastUpgradeKymaOperationIDFromOrchestration(opResponse)
	require.NoError(t, err)

	suite.FinishUpgradeKymaOperationByReconciler(upgradeKymaOperationID)
	suite.AssertClusterConfigWithKubeconfig(opID)

	_, err = suite.db.Operations().GetUpgradeKymaOperationByID(upgradeKymaOperationID)
	require.NoError(t, err)

	// ensure component list after upgrade didn't get changed
	i, err = suite.db.Instances().GetByID(id)
	assert.NoError(t, err, "getting instance after upgrade")
	assert.True(t, i.InstanceDetails.SCMigrationTriggered, "instance SCMigrationTriggered after upgrade")
	rsu2, err := suite.db.RuntimeStates().GetLatestWithReconcilerInputByRuntimeID(i.RuntimeID)
	assert.NoError(t, err, "getting runtime after upgrade")
	assert.NotEqual(t, rsu1.ID, rsu2.ID, "runtime_state ID from update should differ runtime_state ID from upgrade")
	assert.Equal(t, upgradeKymaOperationID, rsu2.OperationID, "runtime state upgrade operation ID")
	assert.ElementsMatch(t, componentNames(rsu2.ClusterSetup.KymaConfig.Components), []string{"ory", "monitoring", "btp-operator"})

	assert.Equal(t, rsu1.ClusterConfig.Name, rsu2.ClusterConfig.Name)
}

/* test disabled due to flakiness
func TestKymaUpgrade_UpgradeAfterMigrationWithNetworkPolicy(t *testing.T) {
	// given
	suite := NewBrokerSuiteTest(t, "2.0")
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
		"name": "testing-cluster",
		"kymaVersion": "2.0.0-rc4"
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
	suite.AssertDisabledNetworkFilterForProvisioning(nil)
	assert.Equal(suite.t, "CUSTOMER", *instance2.Parameters.ErsContext.LicenseType)

	// run upgrade
	orchestrationResp := suite.CallAPI("POST", "upgrade/kyma", `
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
	"kyma": {
		"version": "2.0.0"
	}
}`)
	oID := suite.DecodeOrchestrationID(orchestrationResp)
	suite.AssertReconcilerStartedReconcilingWhenUpgrading(id)
	opResponse := suite.CallAPI("GET", fmt.Sprintf("orchestrations/%s/operations", oID), "")
	upgradeKymaOperationID, err := suite.DecodeLastUpgradeKymaOperationIDFromOrchestration(opResponse)
	require.NoError(t, err)

	suite.AssertReconcilerStartedReconcilingWhenUpgrading(id)
	suite.WaitForOperationState(upgradeKymaOperationID, domain.InProgress)
	suite.FinishUpgradeKymaOperationByReconciler(upgradeKymaOperationID)
	suite.WaitForOperationState(upgradeKymaOperationID, domain.Succeeded)
	suite.AssertClusterConfigWithKubeconfig(opID)

	_, err = suite.db.Operations().GetUpgradeKymaOperationByID(upgradeKymaOperationID)
	require.NoError(t, err)

	// ensure component list after upgrade didn't get changed
	i, err = suite.db.Instances().GetByID(id)
	assert.NoError(t, err, "getting instance after upgrade")
	assert.True(t, i.InstanceDetails.SCMigrationTriggered, "instance SCMigrationTriggered after upgrade")
	rsu2, err := suite.db.RuntimeStates().GetLatestWithReconcilerInputByRuntimeID(i.RuntimeID)
	assert.NoError(t, err, "getting runtime after upgrade")
	assert.NotEqual(t, rsu1.ID, rsu2.ID, "runtime_state ID from update should differ runtime_state ID from upgrade")
	assert.Equal(t, upgradeKymaOperationID, rsu2.OperationID, "runtime state upgrade operation ID")
	assert.ElementsMatch(t, componentNames(rsu2.ClusterSetup.KymaConfig.Components), []string{"ory", "monitoring", "btp-operator"})

	assert.Equal(t, rsu1.ClusterConfig.Name, rsu2.ClusterConfig.Name)

	// ensure license type still persisted
	assert.Equal(suite.t, "CUSTOMER", *instance2.Parameters.ErsContext.LicenseType)
}
*/
