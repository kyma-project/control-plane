package main

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

func TestOrchestration_OneRuntimeHappyPath(t *testing.T) {
	// given
	suite := NewOrchestrationSuite(t)
	runtimeID := suite.CreateProvisionedRuntime(RuntimeOptions{})
	otherRuntimeID := suite.CreateProvisionedRuntime(RuntimeOptions{})
	orchestrationID := suite.CreateOrchestration(runtimeID)

	suite.WaitForOrchestrationState(orchestrationID, internal.InProgress)

	// when
	suite.FinishUpgradeOperationByProvisioner(runtimeID)

	// then
	suite.WaitForOrchestrationState(orchestrationID, internal.Succeeded)

	suite.AssertRuntimeUpgraded(runtimeID)
	suite.AssertRuntimeNotUpgraded(otherRuntimeID)
}
