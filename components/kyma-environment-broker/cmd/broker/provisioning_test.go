//build provisioning-test
package main

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v8/domain"
)

const (
	workersAmount int = 5
)

func TestProvisioning_HappyPath(t *testing.T) {
	// given
	suite := NewProvisioningSuite(t)

	// when
	provisioningOperationID := suite.CreateProvisioning(RuntimeOptions{})

	// then
	suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
	suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

	// when
	suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)
	// simulate the installed fresh Kyma sets the proper label in the Director
	suite.MarkDirectorWithConsoleURL(provisioningOperationID)

	// then
	suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
	suite.AssertAllStagesFinished(provisioningOperationID)
	suite.AssertProvisioningRequest()
}

func TestProvisioningWithReconciler_HappyPath(t *testing.T) {
	t.Skip("not implemented yet")

	// given
	suite := NewBrokerSuiteTest(t)
	defer suite.TearDown()
	iid := uuid.New().String()

	// when
	resp := suite.CallAPI("PUT", fmt.Sprintf("oauth/cf-eu10/v2/service_instances/%s?accepts_incomplete=true", iid),
		`{
					"service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
					"plan_id": "5cb3d976-b85c-42ea-a636-79cadda109a9",
					"context": {
						"sm_platform_credentials": {
							"url": "https://sm.url",
							"credentials": {}
						},
						"globalaccount_id": "g-account-id",
						"subaccount_id": "sub-id",
						"user_id": "john.smith@email.com"
					},
					"globalaccount_id": "g-account-id",
					"subaccount_id": "sub-id",
					"user_id": "john.smith@email.com"
				},
				"parameters": {
					"name": "testing-cluster",
					"oidc": {
						"clientID": "id-initial",
						"signingAlgs": ["xxx"],
						"issuerURL": "https://issuer.url.com"
					}
				}
		}`)

	opID := suite.DecodeOperationID(resp)
	suite.processReconcilingByOperationID(opID)

	// then
	//suite.AssertProvisionRuntimeInput()
	//suite.AssertClusterConfig()
	//suite.AssertClusterState()
}

func TestProvisioning_ClusterParameters(t *testing.T) {
	for tn, tc := range map[string]struct {
		planID           string
		platformRegion   string
		platformProvider internal.CloudProvider
		zonesCount       *int
		region           string

		expectedProfile                    gqlschema.KymaProfile
		expectedProvider                   string
		expectedMinimalNumberOfNodes       int
		expectedMaximumNumberOfNodes       int
		expectedMachineType                string
		expectedSharedSubscription         bool
		expectedSubsciptionHyperscalerType hyperscaler.Type
	}{
		"Regular trial": {
			planID: broker.TrialPlanID,

			expectedMinimalNumberOfNodes:       1,
			expectedMaximumNumberOfNodes:       1,
			expectedMachineType:                "Standard_D4_v3",
			expectedProfile:                    gqlschema.KymaProfileEvaluation,
			expectedProvider:                   "azure",
			expectedSharedSubscription:         true,
			expectedSubsciptionHyperscalerType: hyperscaler.Azure,
		},
		"Freemium aws": {
			planID:           broker.FreemiumPlanID,
			platformProvider: internal.AWS,

			expectedMinimalNumberOfNodes:       1,
			expectedMaximumNumberOfNodes:       1,
			expectedProfile:                    gqlschema.KymaProfileEvaluation,
			expectedProvider:                   "aws",
			expectedSharedSubscription:         false,
			expectedMachineType:                "m5.xlarge",
			expectedSubsciptionHyperscalerType: hyperscaler.AWS,
		},
		"Freemium azure": {
			planID:           broker.FreemiumPlanID,
			platformProvider: internal.Azure,

			expectedMinimalNumberOfNodes:       1,
			expectedMaximumNumberOfNodes:       1,
			expectedProfile:                    gqlschema.KymaProfileEvaluation,
			expectedProvider:                   "azure",
			expectedSharedSubscription:         false,
			expectedMachineType:                "Standard_D4_v3",
			expectedSubsciptionHyperscalerType: hyperscaler.Azure,
		},
		"Production Azure": {
			planID: broker.AzurePlanID,

			expectedMinimalNumberOfNodes:       2,
			expectedMaximumNumberOfNodes:       10,
			expectedMachineType:                "Standard_D8_v3",
			expectedProfile:                    gqlschema.KymaProfileProduction,
			expectedProvider:                   "azure",
			expectedSharedSubscription:         false,
			expectedSubsciptionHyperscalerType: hyperscaler.Azure,
		},
		"HA Azure - provided zonesCount": {
			planID:     broker.AzureHAPlanID,
			zonesCount: ptr.Integer(3),

			expectedMinimalNumberOfNodes:       4,
			expectedMaximumNumberOfNodes:       10,
			expectedMachineType:                "Standard_D4_v3",
			expectedProfile:                    gqlschema.KymaProfileProduction,
			expectedProvider:                   "azure",
			expectedSharedSubscription:         false,
			expectedSubsciptionHyperscalerType: hyperscaler.Azure,
		},
		"HA Azure - default zonesCount": {
			planID: broker.AzureHAPlanID,

			expectedMinimalNumberOfNodes:       4,
			expectedMaximumNumberOfNodes:       10,
			expectedMachineType:                "Standard_D4_v3",
			expectedProfile:                    gqlschema.KymaProfileProduction,
			expectedProvider:                   "azure",
			expectedSharedSubscription:         false,
			expectedSubsciptionHyperscalerType: hyperscaler.Azure,
		},
		"Production AWS": {
			planID: broker.AWSPlanID,

			expectedMinimalNumberOfNodes:       2,
			expectedMaximumNumberOfNodes:       10,
			expectedMachineType:                "m5.2xlarge",
			expectedProfile:                    gqlschema.KymaProfileProduction,
			expectedProvider:                   "aws",
			expectedSharedSubscription:         false,
			expectedSubsciptionHyperscalerType: hyperscaler.AWS,
		},
		"HA AWS - provided zonesCount": {
			planID:     broker.AWSHAPlanID,
			zonesCount: ptr.Integer(3),
			region:     "ap-northeast-2",

			expectedMinimalNumberOfNodes:       4,
			expectedMaximumNumberOfNodes:       10,
			expectedMachineType:                "m5d.xlarge",
			expectedProfile:                    gqlschema.KymaProfileProduction,
			expectedProvider:                   "aws",
			expectedSharedSubscription:         false,
			expectedSubsciptionHyperscalerType: hyperscaler.AWS,
		},
		"HA AWS - default zonesCount": {
			planID: broker.AWSHAPlanID,
			region: "us-west-1",

			expectedMinimalNumberOfNodes:       4,
			expectedMaximumNumberOfNodes:       10,
			expectedMachineType:                "m5d.xlarge",
			expectedProfile:                    gqlschema.KymaProfileProduction,
			expectedProvider:                   "aws",
			expectedSharedSubscription:         false,
			expectedSubsciptionHyperscalerType: hyperscaler.AWS,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			// given
			suite := NewProvisioningSuite(t)

			// when
			provisioningOperationID := suite.CreateProvisioning(RuntimeOptions{
				PlanID:           tc.planID,
				ZonesCount:       tc.zonesCount,
				PlatformRegion:   tc.platformRegion,
				PlatformProvider: tc.platformProvider,
				Region:           tc.region,
			})

			// then
			suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
			suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

			// when
			suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)
			// simulate the installed fresh Kyma sets the proper label in the Director
			suite.MarkDirectorWithConsoleURL(provisioningOperationID)

			// then
			suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
			suite.AssertAllStagesFinished(provisioningOperationID)

			suite.AssertKymaProfile(tc.expectedProfile)
			suite.AssertProvider(tc.expectedProvider)
			suite.AssertMinimalNumberOfNodes(tc.expectedMinimalNumberOfNodes)
			suite.AssertMaximumNumberOfNodes(tc.expectedMaximumNumberOfNodes)
			suite.AssertMachineType(tc.expectedMachineType)
			suite.AssertZonesCount(tc.zonesCount, tc.planID)
			suite.AssertSubscription(tc.expectedSharedSubscription, tc.expectedSubsciptionHyperscalerType)
		})

	}
}

func TestUnsuspensionWithoutShootName(t *testing.T) {
	// given
	suite := NewProvisioningSuite(t)

	// when
	// Create an instance, succeeded suspension operation in the past and a pending unsuspension operation
	unsuspensionOperationID := suite.CreateUnsuspension(RuntimeOptions{})

	// then
	suite.WaitForProvisioningState(unsuspensionOperationID, domain.InProgress)
	suite.AssertProvisionerStartedProvisioning(unsuspensionOperationID)

	// when
	suite.FinishProvisioningOperationByProvisioner(unsuspensionOperationID)
	// simulate the installed fresh Kyma sets the proper label in the Director
	suite.MarkDirectorWithConsoleURL(unsuspensionOperationID)

	// then
	suite.WaitForProvisioningState(unsuspensionOperationID, domain.Succeeded)
	suite.AssertAllStagesFinished(unsuspensionOperationID)
	suite.AssertProvisioningRequest()
}

func TestProvisioning_RuntimeOverrides(t *testing.T) {

	t.Run("should apply overrides to default runtime version", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t)

		// when
		provisioningOperationID := suite.CreateProvisioning(RuntimeOptions{
			OverridesVersion: "1.19",
		})

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
		suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

		// when
		suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)
		// simulate the installed fresh Kyma sets the proper label in the Director
		suite.MarkDirectorWithConsoleURL(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertOverrides(gqlschema.ConfigEntryInput{
			Key:   "foo",
			Value: "bar",
		})
	})

	t.Run("should apply overrides to custom runtime version", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t)

		// when
		provisioningOperationID := suite.CreateProvisioning(RuntimeOptions{
			KymaVersion:      "1.22",
			OverridesVersion: "1.19",
		})

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
		suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

		// when
		suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)
		// simulate the installed fresh Kyma sets the proper label in the Director
		suite.MarkDirectorWithConsoleURL(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertOverrides(gqlschema.ConfigEntryInput{
			Key:   "foo",
			Value: "bar",
		})
	})
}

func TestProvisioning_OIDCValues(t *testing.T) {

	t.Run("should apply default OIDC values when OIDC object is nil", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t)
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
		suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)
		// simulate the installed fresh Kyma sets the proper label in the Director
		suite.MarkDirectorWithConsoleURL(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertOIDC(expectedOIDC)
	})

	t.Run("should apply default OIDC values when all OIDC object's fields are empty", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t)
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
		suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)
		// simulate the installed fresh Kyma sets the proper label in the Director
		suite.MarkDirectorWithConsoleURL(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertOIDC(expectedOIDC)
	})

	t.Run("should apply provided OIDC configuration", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t)
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
		suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)
		// simulate the installed fresh Kyma sets the proper label in the Director
		suite.MarkDirectorWithConsoleURL(provisioningOperationID)

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
		suite := NewProvisioningSuite(t)
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
		suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)
		suite.MarkDirectorWithConsoleURL(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertRuntimeAdmins(expectedAdmins)
	})

	t.Run("should apply new admins list", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t)
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
		suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)
		suite.MarkDirectorWithConsoleURL(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertRuntimeAdmins(expectedAdmins)
	})

	t.Run("should apply empty admin value (list is not empty)", func(t *testing.T) {
		// given
		suite := NewProvisioningSuite(t)
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
		suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)
		suite.MarkDirectorWithConsoleURL(provisioningOperationID)

		// then
		suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
		suite.AssertAllStagesFinished(provisioningOperationID)
		suite.AssertProvisioningRequest()
		suite.AssertRuntimeAdmins(expectedAdmins)
	})
}
