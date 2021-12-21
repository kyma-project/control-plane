package provider

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"
)

func TestGcpTrialInput_ApplyParametersWithRegion(t *testing.T) {
	// given
	svc := GcpTrialInput{
		PlatformRegionMapping: map[string]string{
			"cf-eu": "europe",
		},
	}

	// when
	t.Run("use platform region mapping", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			PlatformRegion: "cf-eu",
		})

		//then
		assert.Equal(t, "europe-west3", input.GardenerConfig.Region)
	})

	// when
	t.Run("use customer mapping", func(t *testing.T) {
		// given
		input := svc.Defaults()
		us := "us"

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			PlatformRegion: "cf-eu",
			Parameters: internal.ProvisioningParametersDTO{
				Region: &us,
			},
		})

		//then
		assert.Equal(t, "us-central1", input.GardenerConfig.Region)
	})

	// when
	t.Run("use default region", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{})

		//then
		assert.Equal(t, "europe-west3", input.GardenerConfig.Region)
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
		assert.Equal(t, "europe-west3", input.GardenerConfig.Region)
	})
}
