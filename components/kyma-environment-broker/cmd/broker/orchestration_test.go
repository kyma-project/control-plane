package main

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
)

func TestKymaUpgrade_OneRuntimeHappyPath(t *testing.T) {
	// given
	suite := NewOrchestrationSuite(t)
	runtimeID := suite.CreateProvisionedRuntime(RuntimeOptions{})
	otherRuntimeID := suite.CreateProvisionedRuntime(RuntimeOptions{})
	orchestrationID := suite.CreateUpgradeKymaOrchestration(runtimeID)

	suite.WaitForOrchestrationState(orchestrationID, orchestration.InProgress)

	// when
	suite.FinishUpgradeOperationByProvisioner(runtimeID)

	// then
	suite.WaitForOrchestrationState(orchestrationID, orchestration.Succeeded)

	suite.AssertRuntimeUpgraded(runtimeID)
	suite.AssertRuntimeNotUpgraded(otherRuntimeID)
}
