//build provisioning-test
package main

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/pivotal-cf/brokerapi/v7/domain"
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

	// then
	suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
	suite.AssertAllStepsFinished(provisioningOperationID)
	suite.AssertDirectorGrafanaTag(provisioningOperationID)
	suite.AssertProvisioningRequest()
}

func TestProvisioning_ClusterParameters(t *testing.T) {
	for tn, tc := range map[string]struct {
		planID string

		expectedProfile              gqlschema.KymaProfile
		expectedProvider             string
		expectedMinimalNumberOfNodes int
		expectedMaximumNumberOfNodes int
		expectedMachineType          string
		expectedSharedSubscription   bool
	}{
		"Regular trial": {
			planID: broker.TrialPlanID,

			expectedMinimalNumberOfNodes: 1,
			expectedMaximumNumberOfNodes: 1,
			expectedMachineType:          "Standard_D4_v3",
			expectedProfile:              gqlschema.KymaProfileEvaluation,
			expectedProvider:             "azure",
			expectedSharedSubscription:   true,
		},
		"Production Azure": {
			planID: broker.AzurePlanID,

			expectedMinimalNumberOfNodes: 2,
			expectedMaximumNumberOfNodes: 10,
			expectedMachineType:          "Standard_D8_v3",
			expectedProfile:              gqlschema.KymaProfileProduction,
			expectedProvider:             "azure",
			expectedSharedSubscription:   false,
		},
		"HA Azure": {
			planID: broker.AzureHAPlanID,

			expectedMinimalNumberOfNodes: 4,
			expectedMaximumNumberOfNodes: 10,
			expectedMachineType:          "Standard_D4_v3",
			expectedProfile:              gqlschema.KymaProfileProduction,
			expectedProvider:             "azure",
			expectedSharedSubscription:   false,
		},
		"Production AWS": {
			planID: broker.AWSPlanID,

			expectedMinimalNumberOfNodes: 2,
			expectedMaximumNumberOfNodes: 10,
			expectedMachineType:          "m5.2xlarge",
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
				PlanID: tc.planID,
			})

			// then
			suite.WaitForProvisioningState(provisioningOperationID, domain.InProgress)
			suite.AssertProvisionerStartedProvisioning(provisioningOperationID)

			// when
			suite.FinishProvisioningOperationByProvisioner(provisioningOperationID)

			// then
			suite.WaitForProvisioningState(provisioningOperationID, domain.Succeeded)
			suite.AssertAllStepsFinished(provisioningOperationID)

			suite.AssertKymaProfile(tc.expectedProfile)
			suite.AssertProvider(tc.expectedProvider)
			suite.AssertMinimalNumberOfNodes(tc.expectedMinimalNumberOfNodes)
			suite.AssertMaximumNumberOfNodes(tc.expectedMaximumNumberOfNodes)
			suite.AssertMachineType(tc.expectedMachineType)
			suite.AssertSharedSubscription(tc.expectedSharedSubscription)
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

	// then
	suite.WaitForProvisioningState(unsuspensionOperationID, domain.Succeeded)
	suite.AssertAllStepsFinished(unsuspensionOperationID)
	suite.AssertDirectorGrafanaTag(unsuspensionOperationID)
	suite.AssertProvisioningRequest()
}
