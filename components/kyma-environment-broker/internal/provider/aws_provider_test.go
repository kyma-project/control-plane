package provider

import (
	"strings"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
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

func TestMultizoneZonesForAWSRegion(t *testing.T) {
	// given
	region := "ap-southeast-1"

	// when
	zones := MultizoneZonesForAWSRegion(region, MultizoneAWSZonesCount)

	// then
	extractedZone := strings.Split(zones, region)[1]
	assert.Equal(t, MultizoneAWSZonesCount, len(extractedZone))
}

func TestAWSHAInput_Defaults(t *testing.T) {
	// given
	svc := AWSHAInput{}

	// when
	input := svc.Defaults()

	// then
	assert.Equal(t, 4, input.GardenerConfig.AutoScalerMin)
	assert.Equal(t, 10, input.GardenerConfig.AutoScalerMax)

	extractedZone := strings.Split(input.GardenerConfig.ProviderSpecificConfig.AwsConfig.Zone, DefaultAWSRegion)[1]
	assert.Equal(t, 2, len(extractedZone))
}
