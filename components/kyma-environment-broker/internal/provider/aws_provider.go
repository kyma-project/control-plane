package provider

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const (
	DefaultAWSRegion      = "eu-central-1"
	DefaultAWSTrialRegion = "eu-west-1"
)

var europeAWS = "eu-west-1"
var usAWS = "us-east-1"
var asiaAWS = "ap-southeast-1"

var toAWSSpecific = map[string]string{
	string(broker.Europe): europeAWS,
	string(broker.Us):     usAWS,
	string(broker.Asia):   asiaAWS,
}

type (
	AWSInput      struct{}
	AWSTrialInput struct {
		PlatformRegionMapping map[string]string
	}
	AWSFreemiumInput struct{}
)

func (p *AWSInput) Defaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       ptr.String("gp2"),
			VolumeSizeGb:   ptr.Integer(50),
			MachineType:    "m5.xlarge",
			Region:         DefaultAWSRegion,
			Provider:       "aws",
			WorkerCidr:     "10.250.0.0/16",
			AutoScalerMin:  3,
			AutoScalerMax:  20,
			MaxSurge:       1,
			MaxUnavailable: 0,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AwsConfig: &gqlschema.AWSProviderConfigInput{
					VpcCidr: "10.250.0.0/16",
					AwsZones: []*gqlschema.AWSZoneInput{
						{
							Name:         ZoneForAWSRegion(DefaultAWSRegion),
							PublicCidr:   "10.250.32.0/20",
							InternalCidr: "10.250.48.0/20",
							WorkerCidr:   "10.250.0.0/19",
						},
					},
				},
			},
		},
	}
}

// awsZones defines a possible suffixes for given AWS regions
// The table is tested in a unit test to check if all necessary regions are covered
var awsZones = map[string]string{
	"eu-central-1":   "abc",
	"eu-west-2":      "abc",
	"ca-central-1":   "abd",
	"sa-east-1":      "abc",
	"us-east-1":      "abcdf",
	"us-west-1":      "ab",
	"ap-northeast-1": "acd",
	"ap-northeast-2": "abc",
	"ap-south-1":     "abc",
	"ap-southeast-1": "abc",
	"ap-southeast-2": "abc",
}

func ZoneForAWSRegion(region string) string {
	zones, found := awsZones[region]
	if !found {
		zones = "a"
	}

	zone := string(zones[rand.Intn(len(zones))])
	return fmt.Sprintf("%s%s", region, zone)
}

func MultipleZonesForAWSRegion(region string, zonesCount int) []string {
	zones, found := awsZones[region]
	if !found {
		zones = "a"
		zonesCount = 1
	}

	availableZones := strings.Split(zones, "")
	rand.Shuffle(len(availableZones), func(i, j int) { availableZones[i], availableZones[j] = availableZones[j], availableZones[i] })
	if zonesCount > len(availableZones) {
		// get maximum number of zones for region
		zonesCount = len(availableZones)
	}

	availableZones = availableZones[:zonesCount]

	var generatedZones []string
	for _, zone := range availableZones {
		generatedZones = append(generatedZones, fmt.Sprintf("%s%s", region, zone))
	}
	return generatedZones
}

func generateMultipleAWSZones(zoneNames []string) []*gqlschema.AWSZoneInput {
	var zones []*gqlschema.AWSZoneInput

	// generate subnets - the subnets in AZ must be inside of the cidr block and non overlapping. example values:
	//vpc:
	//cidr: 10.250.0.0/16
	//zones:
	//	- name: eu-central-1a
	//workers: 10.250.0.0/19
	//public: 10.250.32.0/20
	//internal: 10.250.48.0/20
	//	- name: eu-central-1b
	//workers: 10.250.64.0/19
	//public: 10.250.96.0/20
	//internal: 10.250.112.0/20
	//	- name: eu-central-1c
	//workers: 10.250.128.0/19
	//public: 10.250.160.0/20
	//internal: 10.250.176.0/20
	workerSubnetFmt := "10.250.%d.0/19"
	lbSubnetFmt := "10.250.%d.0/20"
	for i, name := range zoneNames {
		zones = append(zones, &gqlschema.AWSZoneInput{
			Name:         name,
			WorkerCidr:   fmt.Sprintf(workerSubnetFmt, 64*i),
			PublicCidr:   fmt.Sprintf(lbSubnetFmt, 64*i+32),
			InternalCidr: fmt.Sprintf(lbSubnetFmt, 64*i+48),
		})
	}

	return zones
}

func (p *AWSInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	zonesCount := 1
	if pp.Parameters.ZonesCount != nil {
		zonesCount = *pp.Parameters.ZonesCount
	}
	switch {
	// explicit zones list is provided
	case len(pp.Parameters.Zones) > 0:
		input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones = generateMultipleAWSZones(pp.Parameters.Zones)
	// region is provided, with or without zonesCount
	case pp.Parameters.Region != nil && *pp.Parameters.Region != "":
		input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones = generateMultipleAWSZones(MultipleZonesForAWSRegion(*pp.Parameters.Region, zonesCount))
	// region is not provided, zonesCount is provided
	case zonesCount > 1:
		input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones = generateMultipleAWSZones(MultipleZonesForAWSRegion(DefaultAWSRegion, zonesCount))
	}
}

func (p *AWSInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileProduction
}

func (p *AWSInput) Provider() internal.CloudProvider {
	return internal.AWS
}

func (p *AWSTrialInput) Defaults() *gqlschema.ClusterConfigInput {
	return awsLiteDefaults(DefaultAWSTrialRegion)
}

func awsLiteDefaults(region string) *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       ptr.String("gp2"),
			VolumeSizeGb:   ptr.Integer(50),
			MachineType:    "m5.xlarge",
			Region:         region,
			Provider:       "aws",
			WorkerCidr:     "10.250.0.0/19",
			AutoScalerMin:  1,
			AutoScalerMax:  1,
			MaxSurge:       1,
			MaxUnavailable: 0,
			Purpose:        &trialPurpose,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AwsConfig: &gqlschema.AWSProviderConfigInput{
					VpcCidr: "10.250.0.0/16",
					AwsZones: []*gqlschema.AWSZoneInput{
						{
							Name:         ZoneForAWSRegion(region),
							PublicCidr:   "10.250.32.0/20",
							InternalCidr: "10.250.48.0/20",
							WorkerCidr:   "10.250.0.0/19",
						},
					},
				},
			},
		},
	}
}

func (p *AWSTrialInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	params := pp.Parameters

	// read platform region if exists
	if pp.PlatformRegion != "" {
		abstractRegion, found := p.PlatformRegionMapping[pp.PlatformRegion]
		if found {
			r := toAWSSpecific[abstractRegion]
			p.updateRegionWithZones(input, r)
		}
	}

	if params.Region != nil && *params.Region != "" {
		r := toAWSSpecific[*params.Region]
		p.updateRegionWithZones(input, r)
	}
}

func (p *AWSTrialInput) updateRegionWithZones(input *gqlschema.ClusterConfigInput, region string) {
	input.GardenerConfig.Region = region
	input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones[0].Name = ZoneForAWSRegion(region)
}

func (p *AWSTrialInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileEvaluation
}

func (p *AWSTrialInput) Provider() internal.CloudProvider {
	return internal.AWS
}

func (p *AWSFreemiumInput) Defaults() *gqlschema.ClusterConfigInput {
	// Lite (freemium) must have the same defaults as Trial plan, but there was a requirement to change a region only for Trial.
	defaults := awsLiteDefaults(DefaultAWSRegion)

	return defaults
}

func (p *AWSFreemiumInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	if pp.Parameters.Region != nil && *pp.Parameters.Region != "" && pp.Parameters.Zones == nil {
		input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones[0].Name = ZoneForAWSRegion(*pp.Parameters.Region)
	}
}

func (p *AWSFreemiumInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileEvaluation
}

func (p *AWSFreemiumInput) Provider() internal.CloudProvider {
	return internal.AWS
}
