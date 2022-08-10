package provider

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
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

func TestGcpInput_SingleZone_ApplyParameters(t *testing.T) {
	// given
	svc := GcpInput{}

	// when
	t.Run("zones with default region", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{},
		})

		// then
		assert.Equal(t, "europe-west3", input.GardenerConfig.Region)
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, 1)
		assert.Subset(t, []string{"europe-west3-a", "europe-west3-b", "europe-west3-c"}, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones)
	})

	// when
	t.Run("zones with specified region", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Region: ptr.String("us-central1"),
			},
		})

		// then
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, 1)
		assert.Subset(t, []string{"us-central1-a", "us-central1-b", "us-central1-c"}, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones)
	})
}

func TestGcpInput_MultiZone_ApplyParameters(t *testing.T) {
	// given
	svc := GcpInput{MultiZone: true}

	// when
	t.Run("zones with default region", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{},
		})

		// then
		assert.Equal(t, "europe-west3", input.GardenerConfig.Region)
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, 3)
		assert.Subset(t, []string{"europe-west3-a", "europe-west3-b", "europe-west3-c"}, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones)
	})

	// when
	t.Run("zones with specified region", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Region: ptr.String("us-central1"),
			},
		})

		// then
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, 3)
		assert.Subset(t, []string{"us-central1-a", "us-central1-b", "us-central1-c"}, input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones)
	})
}
