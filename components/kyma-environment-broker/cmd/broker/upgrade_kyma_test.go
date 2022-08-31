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
			MachineImage:        ptr.String("coreos"),
			MachineImageVersion: ptr.String("253"),

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
