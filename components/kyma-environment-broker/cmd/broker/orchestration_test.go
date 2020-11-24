package main

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
)

func TestOrchestration_OneRuntimeHappyPath(t *testing.T) {
	// given
	suite := NewOrchestrationSuite(t)
	runtimeID := suite.CreateProvisionedRuntime(RuntimeOptions{})
	otherRuntimeID := suite.CreateProvisionedRuntime(RuntimeOptions{})
	orchestrationID := suite.CreateOrchestration(runtimeID)

	suite.WaitForOrchestrationState(orchestrationID, orchestration.InProgress)

	// when
	suite.FinishUpgradeOperationByProvisioner(runtimeID)

	// then
	suite.WaitForOrchestrationState(orchestrationID, orchestration.Succeeded)

	suite.AssertRuntimeUpgraded(runtimeID)
	suite.AssertRuntimeNotUpgraded(otherRuntimeID)
}
