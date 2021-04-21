//build provisioning-test
package main

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
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

func TestProvisioning_ClusterParameters(t *testing.T) {
	for tn, tc := range map[string]struct {
		planID           string
		platformRegion   string
		platformProvider internal.CloudProvider
		zonesCount       *int

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
		"HA AWS": {
			planID: broker.AWSHAPlanID,

			expectedMinimalNumberOfNodes: 4,
			expectedMaximumNumberOfNodes: 10,
			expectedMachineType:          "m5d.xlarge",
			expectedProfile:              gqlschema.KymaProfileProduction,
			expectedProvider:             "aws",
			expectedSharedSubscription:   false,
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
