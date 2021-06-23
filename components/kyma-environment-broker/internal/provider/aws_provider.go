package provider

import (
	"fmt"
	"math/rand"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const (
	DefaultAWSRegion = "eu-central-1"
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
					Zone:         ZoneForAWSRegion(DefaultAWSRegion),
					VpcCidr:      "10.250.0.0/16",
					PublicCidr:   "10.250.32.0/20",
					InternalCidr: "10.250.48.0/20",
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
	zone := string(zones[rand.Intn(len(zones))])
	return fmt.Sprintf("%s%s", region, zone)
}

func (p *AWSInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	if pp.Parameters.Region != nil && pp.Parameters.Zones == nil {
		input.GardenerConfig.ProviderSpecificConfig.AwsConfig.Zone = ZoneForAWSRegion(*pp.Parameters.Region)
	}
}

func (p *AWSInput) Profile() gqlschema.KymaProfile {
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
			MaxUnavailable: 0,
			Purpose:        &trialPurpose,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AwsConfig: &gqlschema.AWSProviderConfigInput{
					Zone:         ZoneForAWSRegion(DefaultAWSRegion),
					VpcCidr:      "10.250.0.0/16",
					PublicCidr:   "10.250.32.0/20",
					InternalCidr: "10.250.48.0/20",
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
