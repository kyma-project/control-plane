package provider

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"
)

func TestAzureTrialInput_ApplyParametersWithRegion(t *testing.T) {
	// given
	svc := AzureTrialInput{
		PlatformRegionMapping: map[string]string{
			"cf-asia": "asia",
		},
	}

	// when
	t.Run("use platform region mapping", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			PlatformRegion: "cf-asia",
		})

		//then
		assert.Equal(t, "southeastasia", input.GardenerConfig.Region)
	})

	// when
	t.Run("use customer mapping", func(t *testing.T) {
		// given
		input := svc.Defaults()
		us := "us"

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			PlatformRegion: "cf-asia",
			Parameters: internal.ProvisioningParametersDTO{
				Region: &us,
			},
		})

		//then
		assert.Equal(t, "eastus", input.GardenerConfig.Region)
	})

	// when
	t.Run("use default region", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{})

		//then
		assert.Equal(t, DefaultAzureRegion, input.GardenerConfig.Region)
	})

	// when
	t.Run("use random zone", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{})

		zone, err := strconv.Atoi(input.GardenerConfig.ProviderSpecificConfig.AzureConfig.Zones[0])
		require.NoError(t, err)

		//then
		assert.LessOrEqual(t, zone, 3)
		assert.GreaterOrEqual(t, zone, 1)
	})

	// when
	t.Run("use default region for not defined mapping", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			PlatformRegion: "cf-southamerica",
		})

		//then
		assert.Equal(t, DefaultAzureRegion, input.GardenerConfig.Region)
	})
}

func TestAzureHAInput_Defaults(t *testing.T) {
	// given
	svc := AzureHAInput{}

	// when
	input := svc.Defaults()

	// then
	assert.Equal(t, 1, input.GardenerConfig.AutoScalerMin)
	assert.Equal(t, 10, input.GardenerConfig.AutoScalerMax)
	assert.Equal(t, 3, len(input.GardenerConfig.ProviderSpecificConfig.AzureConfig.Zones))
}
