//build provisioning-test
package main

import (
	"testing"

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

}
