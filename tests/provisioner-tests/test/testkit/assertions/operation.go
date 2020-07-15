package assertions

import (
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func AssertOperationSucceed(t *testing.T, expectedType gqlschema.OperationType, expectedRuntimeId string, operation gqlschema.OperationStatus) {
	AssertOperation(t, gqlschema.OperationStateSucceeded, expectedType, expectedRuntimeId, operation)
}

func AssertOperationInProgress(t *testing.T, expectedType gqlschema.OperationType, expectedRuntimeId string, operation gqlschema.OperationStatus) {
	AssertOperation(t, gqlschema.OperationStateInProgress, expectedType, expectedRuntimeId, operation)
}

func AssertOperation(t *testing.T, expectedState gqlschema.OperationState, expectedType gqlschema.OperationType, expectedRuntimeId string, operation gqlschema.OperationStatus) {
	require.NotNil(t, operation.ID)
	require.NotNil(t, operation.Message)

	logrus.Infof("Asserting operation %s is in %s state.", *operation.ID, expectedState)
	logrus.Infof("Operation message: %s", *operation.Message)
	require.Equal(t, expectedState, operation.State)
	assert.Equal(t, expectedType, operation.Operation)
	AssertNotNilAndEqualString(t, expectedRuntimeId, operation.RuntimeID)
}

func AssertUpgradedClusterState(t *testing.T, expected gqlschema.GardenerUpgradeInput, actual gqlschema.GardenerConfig) {
	assert.Equal(t, expected.KubernetesVersion, actual.KubernetesVersion)
	assert.Equal(t, expected.MachineType, actual.MachineType)
	assert.Equal(t, expected.DiskType, actual.DiskType)
	assert.Equal(t, expected.VolumeSizeGb, actual.VolumeSizeGb)
	assert.Equal(t, expected.AutoScalerMin, actual.AutoScalerMin)
	assert.Equal(t, expected.AutoScalerMax, actual.AutoScalerMax)
	assert.Equal(t, expected.MaxSurge, actual.MaxSurge)
	assert.Equal(t, expected.MaxUnavailable, actual.MaxUnavailable)
	assert.Equal(t, expected.Purpose, actual.Purpose)
	assert.Equal(t, expected.EnableKubernetesVersionAutoUpdate, actual.EnableKubernetesVersionAutoUpdate)
	assert.Equal(t, expected.EnableMachineImageVersionAutoUpdate, actual.EnableMachineImageVersionAutoUpdate)
	assert.Equal(t, expected.ProviderSpecificConfig, actual.ProviderSpecificConfig)
}
