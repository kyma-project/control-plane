package main

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/pivotal-cf/brokerapi/v8/domain"
)

func TestKymaDeprovision(t *testing.T) {
	// given
	runtimeOptions := RuntimeOptions{
		GlobalAccountID: globalAccountID,
		SubAccountID:    subAccountID,
		Provider:        internal.AWS,
	}

	suite := NewDeprovisioningSuite(t)
	instanceId := suite.CreateProvisionedRuntime(runtimeOptions)

	// when
	deprovisioningOperationID := suite.CreateDeprovisioning(deprovisioningOpID, instanceId)

	// then
	suite.WaitForDeprovisioningState(deprovisioningOperationID, domain.InProgress)
	suite.AssertProvisionerStartedDeprovisioning(deprovisioningOperationID)

	// when
	suite.FinishDeprovisioningOperationByProvisioner(deprovisioningOperationID)

	// then
	suite.WaitForDeprovisioningState(deprovisioningOperationID, domain.Succeeded)
	suite.AssertInstanceRemoved(instanceId)
}

func TestKymaReDeprovisionSucceeded(t *testing.T) {
	// given
	runtimeOptions := RuntimeOptions{
		GlobalAccountID: globalAccountID,
		SubAccountID:    badSubAccountID,
		Provider:        internal.AWS,
	}

	suite := NewDeprovisioningSuite(t)
	instanceId := suite.CreateProvisionedRuntime(runtimeOptions)

	// when
	deprovisioningOperationID := suite.CreateDeprovisioning(deprovisioningOpID, instanceId)

	// then
	suite.WaitForDeprovisioningState(deprovisioningOperationID, domain.InProgress)
	suite.AssertProvisionerStartedDeprovisioning(deprovisioningOperationID)

	// when
	suite.FinishDeprovisioningOperationByProvisioner(deprovisioningOperationID)

	// then
	suite.WaitForDeprovisioningState(deprovisioningOperationID, domain.Succeeded)
	suite.AssertInstanceNotRemoved(instanceId)

	// when
	runtimeOptions.SubAccountID = subAccountID
	suite.updateSubAccountIDForDeprovisioningOperation(runtimeOptions, instanceId)
	reDeprovisioningOperationID := suite.CreateDeprovisioning(reDeprovisioningOpID, instanceId)

	// then
	suite.WaitForDeprovisioningState(reDeprovisioningOperationID, domain.InProgress)
	suite.AssertProvisionerStartedDeprovisioning(reDeprovisioningOperationID)

	// when
	suite.FinishDeprovisioningOperationByProvisioner(reDeprovisioningOperationID)

	// then
	suite.WaitForDeprovisioningState(reDeprovisioningOperationID, domain.Succeeded)
	suite.AssertInstanceRemoved(instanceId)
}

func TestKymaReDeprovisionFailed(t *testing.T) {
	// given
	runtimeOptions := RuntimeOptions{
		GlobalAccountID: globalAccountID,
		SubAccountID:    badSubAccountID,
		Provider:        internal.AWS,
	}

	suite := NewDeprovisioningSuite(t)
	instanceId := suite.CreateProvisionedRuntime(runtimeOptions)
	// when
	deprovisioningOperationID := suite.CreateDeprovisioning(deprovisioningOpID, instanceId)

	// then
	suite.WaitForDeprovisioningState(deprovisioningOperationID, domain.InProgress)
	suite.AssertProvisionerStartedDeprovisioning(deprovisioningOperationID)

	// when
	suite.FinishDeprovisioningOperationByProvisioner(deprovisioningOperationID)

	// then
	suite.WaitForDeprovisioningState(deprovisioningOperationID, domain.Succeeded)
	suite.AssertInstanceNotRemoved(instanceId)

	// when
	reDeprovisioningOperationID := suite.CreateDeprovisioning(reDeprovisioningOpID, instanceId)

	// then
	suite.WaitForDeprovisioningState(reDeprovisioningOperationID, domain.InProgress)
	suite.AssertProvisionerStartedDeprovisioning(reDeprovisioningOperationID)

	// when
	suite.FinishDeprovisioningOperationByProvisioner(reDeprovisioningOperationID)

	// then
	suite.WaitForDeprovisioningState(reDeprovisioningOperationID, domain.Succeeded)
	suite.AssertInstanceNotRemoved(instanceId)
}
