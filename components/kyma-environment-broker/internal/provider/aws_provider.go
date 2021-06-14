package provider

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const (
	DefaultAWSRegion       = "eu-central-1"
	DefaultAWSHAZonesCount = 2
)

var europeAWS = "eu-central-1"
var usAWS = "us-east-1"
var asiaAWS = "ap-southeast-1"

var toAWSSpecific = map[string]string{
	string(broker.Europe): europeAWS,
	string(broker.Us):     usAWS,
	string(broker.Asia):   asiaAWS,
}

type (
	AWSInput      struct{}
	AWSHAInput    struct{}
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
			MachineType:    "m5.2xlarge",
			Region:         DefaultAWSRegion,
			Provider:       "aws",
			WorkerCidr:     "10.250.0.0/19",
			AutoScalerMin:  2,
			AutoScalerMax:  10,
			MaxSurge:       4,
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
	"us-east-1":      "abcdef",
	"us-west-1":      "ac",
	"ap-northeast-1": "acd",
	"ap-northeast-2": "abcd",
	"ap-south-1":     "ab",
	"ap-southeast-1": "abc",
	"ap-southeast-2": "abc",
}

func ZoneForAWSRegion(region string) string {
	zones, found := awsZones[region]
	if !found {
		zones = "a"
	}
	rand.Seed(time.Now().UnixNano())
	zone := string(zones[rand.Intn(len(zones))])
	return fmt.Sprintf("%s%s", region, zone)
}

func MultipleZonesForAWSRegion(region string, zonesCount int) []string {
	zones, found := awsZones[region]
	if !found {
		zones = "a"
		zonesCount = 1
	}
	rand.Seed(time.Now().UnixNano())

	availableZones := strings.Split(zones, "")
	rand.Shuffle(len(availableZones), func(i, j int) { availableZones[i], availableZones[j] = availableZones[j], availableZones[i] })
	availableZones = availableZones[:zonesCount]

	var generatedZones []string
	for _, zone := range availableZones {
		generatedZones = append(generatedZones, fmt.Sprintf("%s%s", region, zone))
	}
	return generatedZones
}

func generateMultipleAWSZones(region string, zonesCount int) []*gqlschema.AWSZoneInput {
	generatedZones := MultipleZonesForAWSRegion(region, zonesCount)
	var zones []*gqlschema.AWSZoneInput

	// generate subnets - the subnets in AZ must be inside of the cidr block and non overlapping. example values:
	//vpc:
	//cidr: 10.250.0.0/16
	//zones:
	//	- name: eu-central-1a
	//workers: 10.250.0.0/22
	//public: 10.250.20.0/22
	//internal: 10.250.40.0/22
	//	- name: eu-central-1b
	//workers: 10.250.4.0/22
	//public: 10.250.24.0/22
	//internal: 10.250.44.0/22
	//	- name: eu-central-1c
	//workers: 10.250.8.0/22
	//public: 10.250.28.0/22
	//internal: 10.250.48.0/22
	subnetFmt := "10.250.%d.0/22"
	for i, genZone := range generatedZones {
		zones = append(zones, &gqlschema.AWSZoneInput{
			Name:         genZone,
			PublicCidr:   fmt.Sprintf(subnetFmt, 4*i+20),
			InternalCidr: fmt.Sprintf(subnetFmt, 4*i+40),
			WorkerCidr:   fmt.Sprintf(subnetFmt, 4*i),
		})
	}

	return zones
}

func (p *AWSInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	if pp.Parameters.Region != nil && pp.Parameters.Zones == nil {
		input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones[0].Name = ZoneForAWSRegion(*pp.Parameters.Region)
	}
}

func (p *AWSInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileProduction
}

func (p *AWSHAInput) Provider() internal.CloudProvider {
	return internal.AWS
}

func (p *AWSHAInput) Defaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       ptr.String("gp2"),
			VolumeSizeGb:   ptr.Integer(50),
			MachineType:    "m5d.xlarge",
			Region:         DefaultAWSRegion,
			Provider:       "aws",
			WorkerCidr:     "10.250.0.0/19",
			AutoScalerMin:  4,
			AutoScalerMax:  10,
			MaxSurge:       4,
			MaxUnavailable: 0,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AwsConfig: &gqlschema.AWSProviderConfigInput{
					VpcCidr:  "10.250.0.0/16",
					AwsZones: generateMultipleAWSZones(DefaultAWSRegion, DefaultAWSHAZonesCount),
				},
			},
		},
	}
}

func (p *AWSHAInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	if pp.Parameters.Region != nil && pp.Parameters.Zones == nil {
		if pp.Parameters.ZonesCount != nil {
			input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones = generateMultipleAWSZones(*pp.Parameters.Region, *pp.Parameters.ZonesCount)
			return
		}
		input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones = generateMultipleAWSZones(*pp.Parameters.Region, DefaultAzureHAZonesCount)
	}
}

func (p *AWSHAInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileProduction
}

func (p *AWSInput) Provider() internal.CloudProvider {
	return internal.AWS
}

func (p *AWSTrialInput) Defaults() *gqlschema.ClusterConfigInput {
	return awsTrialDefaults()
}

func awsTrialDefaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       ptr.String("gp2"),
			VolumeSizeGb:   ptr.Integer(50),
			MachineType:    "m5.xlarge",
			Region:         DefaultAWSRegion,
			Provider:       "aws",
			WorkerCidr:     "10.250.0.0/19",
			AutoScalerMin:  1,
			AutoScalerMax:  1,
			MaxSurge:       1,
			MaxUnavailable: 1,
			Purpose:        &trialPurpose,
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

func (p *AWSTrialInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	params := pp.Parameters

	// read platform region if exists
	if pp.PlatformRegion != "" {
		abstractRegion, found := p.PlatformRegionMapping[pp.PlatformRegion]
		if found {
			r := toAWSSpecific[abstractRegion]
			input.GardenerConfig.Region = r
		}
	}

	if params.Region != nil {
		input.GardenerConfig.Region = toAWSSpecific[*params.Region]
	}
}

func (p *AWSTrialInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileEvaluation
}

func (p *AWSTrialInput) Provider() internal.CloudProvider {
	return internal.AWS
}

func (p *AWSFreemiumInput) Defaults() *gqlschema.ClusterConfigInput {
	return awsTrialDefaults()
}

func (p *AWSFreemiumInput) ApplyParameters(input *gqlschema.ClusterConfigInput, params internal.ProvisioningParameters) {
	// todo: consider regions
}

func (p *AWSFreemiumInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileEvaluation
}

func (p *AWSFreemiumInput) Provider() internal.CloudProvider {
	return internal.AWS
}
