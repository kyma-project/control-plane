package provider

import (
	"fmt"
	"math/big"
	"math/rand"
	"net/netip"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const (
	DefaultAWSRegion         = "eu-central-1"
	DefaultAWSTrialRegion    = "eu-west-1"
	DefaultEuAccessAWSRegion = "eu-central-1"
	DefaultAWSMultiZoneCount = 3
	DefaultNodesCIDR         = "10.250.0.0/22"
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
	AWSInput struct {
		MultiZone                    bool
		ControlPlaneFailureTolerance string
	}
	AWSTrialInput struct {
		PlatformRegionMapping map[string]string
	}
	AWSFreemiumInput struct{}
)

func (p *AWSInput) Defaults() *gqlschema.ClusterConfigInput {
	zonesCount := 1
	if p.MultiZone {
		zonesCount = DefaultAWSMultiZoneCount
	}
	var controlPlaneFailureTolerance *string = nil
	if p.ControlPlaneFailureTolerance != "" {
		controlPlaneFailureTolerance = &p.ControlPlaneFailureTolerance
	}
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       ptr.String("gp2"),
			VolumeSizeGb:   ptr.Integer(50),
			MachineType:    "m5.xlarge",
			Region:         DefaultAWSRegion,
			Provider:       "aws",
			WorkerCidr:     DefaultNodesCIDR,
			AutoScalerMin:  3,
			AutoScalerMax:  20,
			MaxSurge:       zonesCount,
			MaxUnavailable: 0,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AwsConfig: &gqlschema.AWSProviderConfigInput{
					VpcCidr:  DefaultNodesCIDR,
					AwsZones: generateAWSZones(DefaultNodesCIDR, MultipleZonesForAWSRegion(DefaultAWSRegion, zonesCount)),
				},
			},
			ControlPlaneFailureTolerance: controlPlaneFailureTolerance,
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

/*
*
generateAWSZones - creates a list of AWSZoneInput objects which contains a proper IP ranges.
It generates subnets - the subnets in AZ must be inside of the cidr block and non overlapping. example values:
cidr: 10.250.0.0/16
  - name: eu-central-1a
    workers: 10.250.0.0/18
    public: 10.250.32.0/20
    internal: 10.250.48.0/20
  - name: eu-central-1b
    workers: 10.250.64.0/18
    public: 10.250.96.0/20
    internal: 10.250.112.0/20
  - name: eu-central-1c
    workers: 10.250.128.0/19
    public: 10.250.160.0/20
    internal: 10.250.176.0/20
*/
func generateAWSZones(workerCidr string, zoneNames []string) []*gqlschema.AWSZoneInput {
	var zones []*gqlschema.AWSZoneInput

	cidr, _ := netip.ParsePrefix(workerCidr)
	workerPrefixLength := cidr.Bits() + 2
	workerPrefix, _ := cidr.Addr().Prefix(workerPrefixLength)

	// delta - it is the difference between "public" and "internal" CIDRs, for example:
	//    WorkerCidr:   "10.250.0.0/18",
	//    PublicCidr:   "10.250.32.0/20",
	//    InternalCidr: "10.250.48.0/20",
	// 4 * delta  - difference between two worker (zone) CIDRs
	delta := big.NewInt(1)
	delta.Lsh(delta, uint(30-workerPrefixLength))

	// base - it is an integer, which is based on IP bytes
	base := new(big.Int).SetBytes(workerPrefix.Addr().AsSlice())

	for _, name := range zoneNames {
		zoneWorkerIP, _ := netip.AddrFromSlice(base.Bytes())
		zoneWorkerCidr := netip.PrefixFrom(zoneWorkerIP, workerPrefixLength)

		base.Add(base, delta)
		base.Add(base, delta)
		publicIP, _ := netip.AddrFromSlice(base.Bytes())
		public := netip.PrefixFrom(publicIP, workerPrefixLength+2)

		base.Add(base, delta)
		internalIP, _ := netip.AddrFromSlice(base.Bytes())
		internalPrefix := netip.PrefixFrom(internalIP, workerPrefixLength+2)

		zones = append(zones, &gqlschema.AWSZoneInput{
			Name:         name,
			WorkerCidr:   zoneWorkerCidr.String(),
			PublicCidr:   public.String(),
			InternalCidr: internalPrefix.String(),
		})

		base.Add(base, delta)
	}

	return zones
}

func (p *AWSInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	workerCidr := updateAWSWithWorkerCidr(input, pp)
	zonesCount := 1
	if p.MultiZone {
		zonesCount = DefaultAWSMultiZoneCount
	}
	if len(pp.Parameters.Zones) > 0 {
		zonesCount = len(pp.Parameters.Zones)
	}

	// if the region is provided, override the default one
	if pp.Parameters.Region != nil && *pp.Parameters.Region != "" {
		input.GardenerConfig.Region = *pp.Parameters.Region
	}

	// if the platformRegion is "EU Access" - switch the region to the eu-access
	if internal.IsEuAccess(pp.PlatformRegion) {
		input.GardenerConfig.Region = DefaultEuAccessAWSRegion
	}

	zones := pp.Parameters.Zones
	// if zones are not provided (in the request) - generate it
	if len(zones) == 0 {
		zones = MultipleZonesForAWSRegion(input.GardenerConfig.Region, zonesCount)
	}
	// fill the input with proper zones having Worker CIDR
	input.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones = generateAWSZones(workerCidr, zones)
}

func updateAWSWithWorkerCidr(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) string {
	workerCidr := DefaultNodesCIDR
	if pp.Parameters.Networking != nil {
		workerCidr = pp.Parameters.Networking.NodesCidr
	}
	input.GardenerConfig.WorkerCidr = workerCidr
	input.GardenerConfig.ProviderSpecificConfig.AwsConfig.VpcCidr = workerCidr
	return workerCidr
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
			WorkerCidr:     DefaultNodesCIDR,
			AutoScalerMin:  1,
			AutoScalerMax:  1,
			MaxSurge:       1,
			MaxUnavailable: 0,
			Purpose:        &trialPurpose,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AwsConfig: &gqlschema.AWSProviderConfigInput{
					VpcCidr:  DefaultNodesCIDR,
					AwsZones: generateAWSZones(DefaultNodesCIDR, MultipleZonesForAWSRegion(region, 1)),
				},
			},
		},
	}
}

func (p *AWSTrialInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	params := pp.Parameters

	if internal.IsEuAccess(pp.PlatformRegion) {
		updateRegionWithZones(input, DefaultEuAccessAWSRegion)
		return
	}

	// read platform region if exists
	if pp.PlatformRegion != "" {
		abstractRegion, found := p.PlatformRegionMapping[pp.PlatformRegion]
		if found {
			r := toAWSSpecific[abstractRegion]
			updateRegionWithZones(input, r)
		}
	}

	if params.Region != nil && *params.Region != "" {
		r := toAWSSpecific[*params.Region]
		updateRegionWithZones(input, r)
	}
}

func updateRegionWithZones(input *gqlschema.ClusterConfigInput, region string) {
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
