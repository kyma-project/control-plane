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
	deprovisioningOperationID := suite.CreateDeprovisioning(instanceId)

	// then
	suite.WaitForDeprovisioningState(deprovisioningOperationID, domain.InProgress)
	suite.AssertProvisionerStartedDeprovisioning(deprovisioningOperationID)

	// when
	suite.FinishDeprovisioningOperationByProvisioner(deprovisioningOperationID)

	// then
	suite.WaitForDeprovisioningState(deprovisioningOperationID, domain.Succeeded)
}
