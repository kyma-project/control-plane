package provider

import (
	"fmt"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
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
	t.Run("forget customer empty region", func(t *testing.T) {
		// given
		input := svc.Defaults()
		r := ""

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Region: &r,
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

		zone := input.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones[0].Name

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

	// when
	t.Run("use default with NAT gateway", func(t *testing.T) {
		// given
		input := svc.Defaults()

		//then
		assert.Equal(t, false, *input.GardenerConfig.ProviderSpecificConfig.AzureConfig.EnableNatGateway)
	})
}

func TestAzureInput_SingleZone_ApplyParameters(t *testing.T) {
	// given
	svc := AzureInput{}

	// when
	t.Run("defaults use one zone with dedicated subnet", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{},
		})

		//then
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones, 1)
		assert.Subset(t, []int{1, 2, 3}, azureZoneNames(input.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones))
		for i, zone := range input.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones {
			assert.Equal(t, fmt.Sprintf("10.250.%d.0/19", 32*i), zone.Cidr)
		}
	})

	// when
	t.Run("use zones parameter", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Zones: []string{"2", "3"},
			},
		})

		//then
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones, 2)
		assert.Equal(t, []int{2, 3}, azureZoneNames(input.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones))
		for i, zone := range input.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones {
			assert.Equal(t, fmt.Sprintf("10.250.%d.0/19", 32*i), zone.Cidr)
		}
	})
}

func TestAzureInput_MultiZone_ApplyParameters(t *testing.T) {
	// given
	svc := AzureInput{MultiZone: true}

	// when
	t.Run("defaults use three zones with dedicated subnet", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{},
		})

		//then
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones, DefaultAzureMultiZoneCount)
		assert.ElementsMatch(t, []int{1, 2, 3}, azureZoneNames(input.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones))
		for i, zone := range input.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones {
			assert.Equal(t, fmt.Sprintf("10.250.%d.0/19", 32*i), zone.Cidr)
		}
	})

	// when
	t.Run("use zones parameter", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Zones: []string{"2", "3"},
			},
		})

		//then
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones, 2)
		assert.Equal(t, []int{2, 3}, azureZoneNames(input.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones))
		for i, zone := range input.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones {
			assert.Equal(t, fmt.Sprintf("10.250.%d.0/19", 32*i), zone.Cidr)
		}
	})
}

func azureZoneNames(zones []*gqlschema.AzureZoneInput) []int {
	zoneNames := []int{}

	for _, zone := range zones {
		zoneNames = append(zoneNames, zone.Name)
	}

	return zoneNames
}
