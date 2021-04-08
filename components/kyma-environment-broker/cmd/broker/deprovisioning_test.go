package main

import (
	"testing"

	"github.com/pivotal-cf/brokerapi/v7/domain"
)

func TestKymaDeprovision(t *testing.T) {
	// given
	runtimeOptions := RuntimeOptions{
		GlobalAccountID: globalAccountID,
		SubAccountID:    subAccountID,
	}

	suite := NewDeprovisioningSuite(t)
	instanceId := suite.CreateProvisionedRuntime(runtimeOptions)

	// when
	deprovisioningOperationID := suite.CreateDeprovisioning(instanceId)

	// then
	suite.WaitForDeprovisioningState(deprovisioningOperationID, domain.InProgress)
	suite.AssertProvisionerStartedDeprovisioning(deprovisioningOperationID)

	// when
	suite.FinishDeprovisioningOperationByProvisioner(deprovisioningOperationID)

	// then
	suite.WaitForDeprovisioningState(deprovisioningOperationID, domain.Succeeded)
	// suite.AssertAllStepsFinished(deprovisioningOperationID)
}
