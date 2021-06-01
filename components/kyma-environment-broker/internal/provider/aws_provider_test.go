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
	// given
	region := "us-east-1"

	// when
	generatedZones := MultipleZonesForAWSRegion(region, DefaultAWSHAZonesCount)

	// then
	for _, zone := range generatedZones {
		regionFromZone := zone[:len(zone)-1]
		assert.Equal(t, region, regionFromZone)
	}
	assert.Equal(t, DefaultAWSHAZonesCount, len(generatedZones))
}

func TestAWSHAInput_ApplyParametersWithRegion(t *testing.T) {
	// given
	svc := AWSHAInput{}

	// when
	t.Run("use default region and default zones count", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{})

		//then
		assert.Equal(t, DefaultAWSRegion, input.GardenerConfig.Region)
		assert.Equal(t, DefaultAzureHAZonesCount, len(input.GardenerConfig.ProviderSpecificConfig.AwsConfig.Zones))
	})

	// when
	t.Run("use default region and zonesCount input parameter", func(t *testing.T) {
		// given
		input := svc.Defaults()
		inputRegion := "us-east-1"
		zonesCount := 4

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				ZonesCount: ptr.Integer(zonesCount),
				Region:     ptr.String(inputRegion),
			},
		})

		//then
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.AwsConfig.Zones, zonesCount)

		for _, zone := range input.GardenerConfig.ProviderSpecificConfig.AwsConfig.Zones {
			regionFromZone := zone.Name[:len(zone.Name)-1]
			assert.Equal(t, inputRegion, regionFromZone)
		}
	})
}

func TestAWSHAInput_Defaults(t *testing.T) {
	// given
	svc := AWSHAInput{}

	// when
	input := svc.Defaults()

	// then
	assert.Equal(t, 4, input.GardenerConfig.AutoScalerMin)
	assert.Equal(t, 10, input.GardenerConfig.AutoScalerMax)
	assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.AwsConfig.Zones, 2)
}
