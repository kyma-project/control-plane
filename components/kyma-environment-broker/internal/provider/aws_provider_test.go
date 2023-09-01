package provider

import (
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/stretchr/testify/assert"
)

func TestAWSZones(t *testing.T) {
	regions := broker.AWSRegions(false)
	for _, region := range regions {
		_, exists := awsZones[region]
		assert.True(t, exists)
	}
	_, exists := awsZones[DefaultAWSRegion]
	assert.True(t, exists)
}

func TestAWSZonesWithCustomNodeIPRange(t *testing.T) {
	svc := AWSInput{
		MultiZone: true,
	}

	clusterConfigInput := svc.Defaults()
	svc.ApplyParameters(clusterConfigInput, internal.ProvisioningParameters{
		Parameters: internal.ProvisioningParametersDTO{
			Networking: &internal.NetworkingDTO{
				NodesCidr: "10.180.0.0/16",
			},
		},
	})

	assert.Equal(t, "10.180.0.0/16", clusterConfigInput.GardenerConfig.WorkerCidr)
	assert.Equal(t, "10.180.0.0/16", clusterConfigInput.GardenerConfig.ProviderSpecificConfig.AwsConfig.VpcCidr)

	for tname, tcase := range map[string]struct {
		givenNodesCidr   string
		expectedAwsZones []gqlschema.AWSZoneInput
	}{
		"Regular 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			expectedAwsZones: []gqlschema.AWSZoneInput{
				{
					WorkerCidr:   "10.250.0.0/19",
					PublicCidr:   "10.250.32.0/20",
					InternalCidr: "10.250.48.0/20",
				},
				{
					WorkerCidr:   "10.250.64.0/19",
					PublicCidr:   "10.250.96.0/20",
					InternalCidr: "10.250.112.0/20",
				},
				{
					WorkerCidr:   "10.250.128.0/19",
					PublicCidr:   "10.250.160.0/20",
					InternalCidr: "10.250.176.0/20",
				},
			},
		},
		"Regular 10.180.0.0/23": {
			givenNodesCidr: "10.180.0.0/23",
			expectedAwsZones: []gqlschema.AWSZoneInput{
				{
					WorkerCidr:   "10.180.0.0/26",
					PublicCidr:   "10.180.0.64/27",
					InternalCidr: "10.180.0.96/27",
				},
				{
					WorkerCidr:   "10.180.0.128/26",
					PublicCidr:   "10.180.0.192/27",
					InternalCidr: "10.180.0.224/27",
				},
				{
					WorkerCidr:   "10.180.1.0/26",
					PublicCidr:   "10.180.1.64/27",
					InternalCidr: "10.180.1.96/27",
				},
			},
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			svc := AWSInput{
				MultiZone: true,
			}

			// when
			clusterConfigInput := svc.Defaults()
			svc.ApplyParameters(clusterConfigInput, internal.ProvisioningParameters{
				Parameters: internal.ProvisioningParametersDTO{
					Networking: &internal.NetworkingDTO{
						NodesCidr: tcase.givenNodesCidr,
					},
				},
			})

			// then
			assert.Equal(t, tcase.givenNodesCidr, clusterConfigInput.GardenerConfig.WorkerCidr)
			assert.Equal(t, tcase.givenNodesCidr, clusterConfigInput.GardenerConfig.ProviderSpecificConfig.AwsConfig.VpcCidr)

			for i, expectedZone := range tcase.expectedAwsZones {
				assertAWSIpRanges(t, expectedZone, clusterConfigInput.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones[i])
			}
		})
	}

}

func assertAWSIpRanges(t *testing.T, zone gqlschema.AWSZoneInput, input *gqlschema.AWSZoneInput) {
	assert.Equal(t, zone.InternalCidr, input.InternalCidr)
	assert.Equal(t, zone.WorkerCidr, input.WorkerCidr)
	assert.Equal(t, zone.PublicCidr, input.PublicCidr)
}

func TestAWSZonesForEuAccess(t *testing.T) {
	regions := broker.AWSRegions(true)
	for _, region := range regions {
		_, exists := awsZones[region]
		assert.True(t, exists)
	}
	_, exists := awsZones[DefaultEuAccessAWSRegion]
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
			var zones []string
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

func TestAWSInput_SingleZone_ApplyParameters(t *testing.T) {
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
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones, 1)

		for _, zone := range input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones {
			regionFromZone := zone.Name[:len(zone.Name)-1]
			assert.Equal(t, DefaultAWSRegion, regionFromZone)
		}
	})

	t.Run("use default region and default zones count for EU Access", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			PlatformRegion: "cf-ch20",
		})

		//then
		assert.Equal(t, DefaultEuAccessAWSRegion, input.GardenerConfig.Region)
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones, 1)

		for _, zone := range input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones {
			regionFromZone := zone.Name[:len(zone.Name)-1]
			assert.Equal(t, DefaultEuAccessAWSRegion, regionFromZone)
		}
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

func TestAWSInput_MultiZone_ApplyParameters(t *testing.T) {
	// given
	svc := AWSInput{MultiZone: true, ControlPlaneFailureTolerance: "zone"}

	// when
	t.Run("use default region and default zones count", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{})

		//then
		assert.Equal(t, DefaultAWSRegion, input.GardenerConfig.Region)
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones, DefaultAWSMultiZoneCount)

		for _, zone := range input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones {
			regionFromZone := zone.Name[:len(zone.Name)-1]
			assert.Equal(t, DefaultAWSRegion, regionFromZone)
		}

		assert.Equal(t, "zone", *input.GardenerConfig.ControlPlaneFailureTolerance)
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
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones, DefaultAWSMultiZoneCount)

		for _, zone := range input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones {
			regionFromZone := zone.Name[:len(zone.Name)-1]
			assert.Equal(t, inputRegion, regionFromZone)
		}
		assert.Equal(t, "zone", *input.GardenerConfig.ControlPlaneFailureTolerance)
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
		assert.Equal(t, "zone", *input.GardenerConfig.ControlPlaneFailureTolerance)
	})
}

func TestAWSTrialInput_ApplyParameters(t *testing.T) {
	t.Run("AWS trial", func(t *testing.T) {
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
	})

	t.Run("AWS trial with EU Access restrictions", func(t *testing.T) {
		svc := AWSTrialInput{PlatformRegionMapping: map[string]string{
			"cf-eu10": "europe",
			"cf-us10": "us",
		}}
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			PlatformRegion: "cf-eu11",
		})

		// then
		assert.Contains(t, DefaultEuAccessAWSRegion, input.GardenerConfig.Region)
	})
}
