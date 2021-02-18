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
	assertWhenExpectedStringNotNil(t, expected.KubernetesVersion, actual.KubernetesVersion)
	assertWhenExpectedStringNotNil(t, expected.MachineType, actual.MachineType)
	assertWhenExpectedStringNotNil(t, expected.DiskType, actual.DiskType)
	assertWhenExpectedIntNotNil(t, expected.VolumeSizeGb, actual.VolumeSizeGb)
	assertWhenExpectedIntNotNil(t, expected.AutoScalerMin, actual.AutoScalerMin)
	assertWhenExpectedIntNotNil(t, expected.AutoScalerMax, actual.AutoScalerMax)
	assertWhenExpectedIntNotNil(t, expected.MaxSurge, actual.MaxSurge)
	assertWhenExpectedIntNotNil(t, expected.MaxUnavailable, actual.MaxUnavailable)
	assertWhenExpectedStringNotNil(t, expected.Purpose, actual.Purpose)
	assertWhenExpectedBoolNotNil(t, expected.EnableKubernetesVersionAutoUpdate, actual.EnableKubernetesVersionAutoUpdate)
	assertWhenExpectedBoolNotNil(t, expected.EnableMachineImageVersionAutoUpdate, actual.EnableMachineImageVersionAutoUpdate)

	if expected.ProviderSpecificConfig != nil {

		if expected.ProviderSpecificConfig.AzureConfig != nil {
			azureConfig, ok := actual.ProviderSpecificConfig.(*gqlschema.AzureProviderConfig)
			require.True(t, ok)

			AssertNotNilAndEqualString(t, expected.ProviderSpecificConfig.AzureConfig.VnetCidr, azureConfig.VnetCidr)
			assert.ElementsMatch(t, expected.ProviderSpecificConfig.AzureConfig.Zones, azureConfig.Zones)
		}

		if expected.ProviderSpecificConfig.AwsConfig != nil {
			awsConfig, ok := actual.ProviderSpecificConfig.(*gqlschema.AWSProviderConfig)
			require.True(t, ok)

			AssertNotNilAndEqualString(t, expected.ProviderSpecificConfig.AwsConfig.VpcCidr, awsConfig.VpcCidr)
			AssertNotNilAndEqualString(t, expected.ProviderSpecificConfig.AwsConfig.Zone, awsConfig.Zone)
			AssertNotNilAndEqualString(t, expected.ProviderSpecificConfig.AwsConfig.InternalCidr, awsConfig.InternalCidr)
			AssertNotNilAndEqualString(t, expected.ProviderSpecificConfig.AwsConfig.PublicCidr, awsConfig.PublicCidr)
		}

		if expected.ProviderSpecificConfig.GcpConfig != nil {
			gcpConfig, ok := actual.ProviderSpecificConfig.(*gqlschema.GCPProviderConfig)
			require.True(t, ok)

			assert.ElementsMatch(t, expected.ProviderSpecificConfig.GcpConfig.Zones, gcpConfig.Zones)
		}
	}
}

func assertWhenExpectedStringNotNil(t *testing.T, expected, actual *string) {
	if expected != nil {
		assert.Equal(t, expected, actual)
	}
}

func assertWhenExpectedIntNotNil(t *testing.T, expected, actual *int) {
	if expected != nil {
		assert.Equal(t, expected, actual)
	}
}

func assertWhenExpectedBoolNotNil(t *testing.T, expected, actual *bool) {
	if expected != nil {
		assert.Equal(t, expected, actual)
	}
}
