package main

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
)

func TestKymaUpgrade_OneRuntimeHappyPath(t *testing.T) {
	// given

	suite := NewOrchestrationSuite(t, nil)
	runtimeID := suite.CreateProvisionedRuntime(RuntimeOptions{})
	otherRuntimeID := suite.CreateProvisionedRuntime(RuntimeOptions{})
	orchestrationParams := fixOrchestrationParams(runtimeID)
	orchestrationID := suite.CreateUpgradeKymaOrchestration(orchestrationParams)

	suite.WaitForOrchestrationState(orchestrationID, orchestration.InProgress)

	// when
	suite.FinishUpgradeOperationByReconciler(runtimeID)

	// then
	suite.WaitForOrchestrationState(orchestrationID, orchestration.Succeeded)

	suite.AssertRuntimeUpgraded(runtimeID, "2.0.3")
	suite.AssertRuntimeNotUpgraded(otherRuntimeID)
}

func TestKymaUpgrade_VersionParameter(t *testing.T) {
	// given
	givenVersion := "2.0.0-rc5"
	suite := NewOrchestrationSuite(t, []string{givenVersion})
	runtimeID := suite.CreateProvisionedRuntime(RuntimeOptions{})
	otherRuntimeID := suite.CreateProvisionedRuntime(RuntimeOptions{})
	orchestrationParams := fixOrchestrationParams(runtimeID)
	orchestrationParams.Kyma.Version = givenVersion
	orchestrationID := suite.CreateUpgradeKymaOrchestration(orchestrationParams)

	suite.WaitForOrchestrationState(orchestrationID, orchestration.InProgress)

	// when
	suite.FinishUpgradeOperationByReconciler(runtimeID)

	// then
	suite.WaitForOrchestrationState(orchestrationID, orchestration.Succeeded)

	suite.AssertRuntimeUpgraded(runtimeID, givenVersion)
	suite.AssertRuntimeNotUpgraded(otherRuntimeID)
}

func TestClusterUpgrade_OneRuntimeHappyPath(t *testing.T) {
	// given
	suite := NewOrchestrationSuite(t, nil)
	runtimeID := suite.CreateProvisionedRuntime(RuntimeOptions{})
	otherRuntimeID := suite.CreateProvisionedRuntime(RuntimeOptions{})
	orchestrationParams := fixOrchestrationParams(runtimeID)
	orchestrationID := suite.CreateUpgradeClusterOrchestration(orchestrationParams)

	suite.WaitForOrchestrationState(orchestrationID, orchestration.InProgress)

	// when
	suite.FinishUpgradeShootOperationByProvisioner(runtimeID)

	// then
	suite.WaitForOrchestrationState(orchestrationID, orchestration.Succeeded)

	suite.AssertShootUpgraded(runtimeID)
	suite.AssertShootNotUpgraded(otherRuntimeID)
}

func fixOrchestrationParams(runtimeID string) orchestration.Parameters {
	return orchestration.Parameters{
		Targets: orchestration.TargetSpec{
			Include: []orchestration.RuntimeTarget{
				{RuntimeID: runtimeID},
			},
		},
		Strategy: orchestration.StrategySpec{
			Type:     orchestration.ParallelStrategy,
			Schedule: orchestration.Immediate,
			Parallel: orchestration.ParallelStrategySpec{Workers: 1},
		},
		DryRun:     false,
		Kubernetes: &orchestration.KubernetesParameters{KubernetesVersion: ""},
		Kyma:       &orchestration.KymaParameters{Version: ""},
	}
}
