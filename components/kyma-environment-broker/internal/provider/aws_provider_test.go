package provider

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/stretchr/testify/assert"
)

func TestAWSZones(t *testing.T) {
	regions := broker.AWSRegions()
	for _, region := range regions {
		_, exists := awsZones[region]
		assert.True(t, exists)
	}
	_, exists := awsZones[DefaultAWSRegion]
	assert.True(t, exists)
}

func TestMultipleZonesForAWSRegion(t *testing.T) {
	t.Run("for valid zonesCount", func(t *testing.T) {
		// given
		region := "us-east-1"

		// when
		generatedZones := MultipleZonesForAWSRegion(region, 3)

		// then
		for _, zone := range generatedZones {
			regionFromZone := zone[:len(zone)-1]
			assert.Equal(t, region, regionFromZone)
		}
		assert.Equal(t, 3, len(generatedZones))
		// check if all zones are unique
		assert.Condition(t, func() (success bool) {
			zones := []string{}
			for _, zone := range generatedZones {
				for _, z := range zones {
					if zone == z {
						return false
					}
				}
				zones = append(zones, zone)
			}
			return true
		})
	})
	t.Run("for zonesCount exceeding maximum zones for region", func(t *testing.T) {
		// given
		region := "us-east-1"
		zonesCountExceedingMaximum := 20
		maximumZonesForRegion := len(awsZones[region])
		// "us-east-1" region has maximum 6 zones, user request 20

		// when
		generatedZones := MultipleZonesForAWSRegion(region, zonesCountExceedingMaximum)

		// then
		for _, zone := range generatedZones {
			regionFromZone := zone[:len(zone)-1]
			assert.Equal(t, region, regionFromZone)
		}
		assert.Equal(t, maximumZonesForRegion, len(generatedZones))
	})
}

func TestAWSInput_ApplyParameters(t *testing.T) {
	// given
	svc := AWSInput{}

	// when
	t.Run("use default region and default zones count", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{})

		//then
		assert.Equal(t, DefaultAWSRegion, input.GardenerConfig.Region)
		assert.Equal(t, 1, len(input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones))
	})

	// when
	t.Run("use region input parameter", func(t *testing.T) {
		// given
		input := svc.Defaults()
		inputRegion := "us-east-1"

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Region: ptr.String(inputRegion),
			},
		})

		//then
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones, 1)

		for _, zone := range input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones {
			regionFromZone := zone.Name[:len(zone.Name)-1]
			assert.Equal(t, inputRegion, regionFromZone)
		}
	})

	// when
	t.Run("use zonesCount input parameters (default region)", func(t *testing.T) {
		// given
		input := svc.Defaults()
		zonesCount := 3

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				ZonesCount: ptr.Integer(zonesCount),
			},
		})

		//then
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones, zonesCount)

		for _, zone := range input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones {
			regionFromZone := zone.Name[:len(zone.Name)-1]
			assert.Equal(t, DefaultAWSRegion, regionFromZone)
		}
	})

	// when
	t.Run("use region and zonesCount input parameters", func(t *testing.T) {
		// given
		input := svc.Defaults()
		inputRegion := "us-east-1"
		zonesCount := 3

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				ZonesCount: ptr.Integer(zonesCount),
				Region:     ptr.String(inputRegion),
			},
		})

		//then
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones, zonesCount)

		for _, zone := range input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones {
			regionFromZone := zone.Name[:len(zone.Name)-1]
			assert.Equal(t, inputRegion, regionFromZone)
		}
	})

	// when
	t.Run("use zones list input parameter", func(t *testing.T) {
		// given
		input := svc.Defaults()
		zones := []string{"eu-central-1a", "eu-central-1b"}

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Zones: zones,
			},
		})

		//then
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones, len(zones))

		for i, zone := range input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones {
			assert.Equal(t, zones[i], zone.Name)
		}
	})
}

func TestAWSTrialInput_ApplyParameters(t *testing.T) {
	// given
	svc := AWSTrialInput{PlatformRegionMapping: map[string]string{
		"cf-eu10": "europe",
		"cf-us10": "us",
	}}
	input := svc.Defaults()

	// when
	svc.ApplyParameters(input, internal.ProvisioningParameters{
		PlatformRegion: "cf-us10",
	})

	// then
	assert.Contains(t, input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones[0].Name, input.GardenerConfig.Region)
}
