package provider

import (
	"math/big"
	"math/rand"
	"net/netip"
	"strconv"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/networking"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const (
	DefaultAzureRegion         = "eastus"
	DefaultEuAccessAzureRegion = "switzerlandnorth"
	DefaultAzureMultiZoneCount = 3
)

var europeAzure = "westeurope"
var usAzure = "eastus"
var asiaAzure = "southeastasia"

var trialPurpose = "evaluation"

var toAzureSpecific = map[string]*string{
	string(broker.Europe): &europeAzure,
	string(broker.Us):     &usAzure,
	string(broker.Asia):   &asiaAzure,
}

type (
	AzureInput struct {
		MultiZone                    bool
		ControlPlaneFailureTolerance string
	}
	AzureLiteInput  struct{}
	AzureTrialInput struct {
		PlatformRegionMapping map[string]string
	}
	AzureFreemiumInput struct{}
)

func (p *AzureInput) Defaults() *gqlschema.ClusterConfigInput {
	zonesCount := 1
	if p.MultiZone {
		zonesCount = DefaultAzureMultiZoneCount
	}
	var controlPlaneFailureTolerance *string = nil
	if p.ControlPlaneFailureTolerance != "" {
		controlPlaneFailureTolerance = &p.ControlPlaneFailureTolerance
	}
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       ptr.String("Standard_LRS"),
			VolumeSizeGb:   ptr.Integer(50),
			MachineType:    "Standard_D4_v3",
			Region:         DefaultAzureRegion,
			Provider:       "azure",
			WorkerCidr:     networking.DefaultNodesCIDR,
			AutoScalerMin:  3,
			AutoScalerMax:  20,
			MaxSurge:       zonesCount,
			MaxUnavailable: 0,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AzureConfig: &gqlschema.AzureProviderConfigInput{
					VnetCidr:         networking.DefaultNodesCIDR,
					AzureZones:       generateAzureZones(networking.DefaultNodesCIDR, generateRandomAzureZones(zonesCount)),
					EnableNatGateway: ptr.Bool(true),
				},
			},
			ControlPlaneFailureTolerance: controlPlaneFailureTolerance,
		},
	}
}

func (p *AzureInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	if internal.IsEuAccess(pp.PlatformRegion) {
		updateString(&input.GardenerConfig.Region, ptr.String(DefaultEuAccessAzureRegion))
		return
	}
	workerCidr := networking.DefaultNodesCIDR
	if pp.Parameters.Networking != nil {
		workerCidr = pp.Parameters.Networking.NodesCidr
	}
	input.GardenerConfig.WorkerCidr = workerCidr
	input.GardenerConfig.ProviderSpecificConfig.AzureConfig.VnetCidr = workerCidr
	zonesCount := 1
	if p.MultiZone {
		zonesCount = DefaultAzureMultiZoneCount
	}

	// explicit zones list is provided
	if len(pp.Parameters.Zones) > 0 {
		var zones []int
		for _, inputZone := range pp.Parameters.Zones {
			zone, err := strconv.Atoi(inputZone)
			if err != nil || zone < 1 || zone > 3 {
				continue
			}
			zones = append(zones, zone)
		}
		input.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones = generateAzureZones(workerCidr, zones)
	} else {
		input.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones = generateAzureZones(workerCidr, generateRandomAzureZones(zonesCount))
	}
}

func (p *AzureInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileProduction
}

func (p *AzureInput) Provider() internal.CloudProvider {
	return internal.Azure
}

func (p *AzureLiteInput) Defaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       ptr.String("Standard_LRS"),
			VolumeSizeGb:   ptr.Integer(50),
			MachineType:    "Standard_D4_v3",
			Region:         DefaultAzureRegion,
			Provider:       "azure",
			WorkerCidr:     networking.DefaultNodesCIDR,
			AutoScalerMin:  2,
			AutoScalerMax:  10,
			MaxSurge:       1,
			MaxUnavailable: 0,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AzureConfig: &gqlschema.AzureProviderConfigInput{
					VnetCidr: networking.DefaultNodesCIDR,
					AzureZones: []*gqlschema.AzureZoneInput{
						{
							Name: generateRandomAzureZone(),
							Cidr: networking.DefaultNodesCIDR,
						},
					},
					EnableNatGateway: ptr.Bool(true),
				},
			},
		},
	}
}

func (p *AzureLiteInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	if internal.IsEuAccess(pp.PlatformRegion) {
		updateString(&input.GardenerConfig.Region, ptr.String(DefaultEuAccessAzureRegion))
	}

	updateAzureSingleNodeWorkerCidr(input, pp)
}

func updateAzureSingleNodeWorkerCidr(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	workerCIDR := networking.DefaultNodesCIDR
	if pp.Parameters.Networking != nil {
		workerCIDR = pp.Parameters.Networking.NodesCidr
	}
	input.GardenerConfig.WorkerCidr = workerCIDR
	input.GardenerConfig.ProviderSpecificConfig.AzureConfig.VnetCidr = workerCIDR
	input.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones[0].Cidr = workerCIDR
}

func (p *AzureLiteInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileEvaluation
}

func (p *AzureLiteInput) Provider() internal.CloudProvider {
	return internal.Azure
}

func (p *AzureTrialInput) Defaults() *gqlschema.ClusterConfigInput {
	return azureTrialDefaults()
}

func azureTrialDefaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       ptr.String("Standard_LRS"),
			VolumeSizeGb:   ptr.Integer(50),
			MachineType:    "Standard_D4_v3",
			Region:         DefaultAzureRegion,
			Provider:       "azure",
			WorkerCidr:     networking.DefaultNodesCIDR,
			AutoScalerMin:  1,
			AutoScalerMax:  1,
			MaxSurge:       1,
			MaxUnavailable: 0,
			Purpose:        &trialPurpose,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AzureConfig: &gqlschema.AzureProviderConfigInput{
					VnetCidr: networking.DefaultNodesCIDR,
					AzureZones: []*gqlschema.AzureZoneInput{
						{
							Name: generateRandomAzureZone(),
							Cidr: networking.DefaultNodesCIDR,
						},
					},
					EnableNatGateway: ptr.Bool(false),
				},
			},
		},
	}
}

func (p *AzureTrialInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	params := pp.Parameters

	if internal.IsEuAccess(pp.PlatformRegion) {
		updateString(&input.GardenerConfig.Region, ptr.String(DefaultEuAccessAzureRegion))
		return
	}

	// read platform region if exists
	if pp.PlatformRegion != "" {
		abstractRegion, found := p.PlatformRegionMapping[pp.PlatformRegion]
		if found {
			r := toAzureSpecific[abstractRegion]
			updateString(&input.GardenerConfig.Region, r)
		}
	}

	if params.Region != nil && *params.Region != "" {
		updateString(&input.GardenerConfig.Region, toAzureSpecific[*params.Region])
	}

	updateAzureSingleNodeWorkerCidr(input, pp)
}

func (p *AzureTrialInput) Provider() internal.CloudProvider {
	return internal.Azure
}

func (p *AzureTrialInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileEvaluation
}

func (p *AzureFreemiumInput) Defaults() *gqlschema.ClusterConfigInput {
	return azureTrialDefaults()
}

func (p *AzureFreemiumInput) ApplyParameters(input *gqlschema.ClusterConfigInput, params internal.ProvisioningParameters) {
	updateSlice(&input.GardenerConfig.ProviderSpecificConfig.AzureConfig.Zones, params.Parameters.Zones)
}

func (p *AzureFreemiumInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileEvaluation
}

func (p *AzureFreemiumInput) Provider() internal.CloudProvider {
	return internal.Azure
}

func generateRandomAzureZone() int {
	const (
		min = 1
		max = 3
	)

	// generates random number from 1-3 range
	getRandomNumber := func() int {
		return rand.Intn(max-min+1) + min
	}

	return getRandomNumber()
}

func generateRandomAzureZones(zonesCount int) []int {
	zones := []int{1, 2, 3}
	if zonesCount > 3 {
		zonesCount = 3
	}

	rand.Shuffle(len(zones), func(i, j int) { zones[i], zones[j] = zones[j], zones[i] })
	return zones[:zonesCount]
}

func generateAzureZones(workerCidr string, zoneNames []int) []*gqlschema.AzureZoneInput {
	var zones []*gqlschema.AzureZoneInput

	cidr, _ := netip.ParsePrefix(workerCidr)
	workerPrefixLength := cidr.Bits() + 3
	workerPrefix, _ := cidr.Addr().Prefix(workerPrefixLength)
	// delta - it is the difference between CIDRs of two zones:
	//    zone1:   "10.250.0.0/19",
	//    zone2:   "10.250.32.0/19",
	delta := big.NewInt(1)
	delta.Lsh(delta, uint(32-workerPrefixLength))

	// zoneIPValue - it is an integer, which is based on IP bytes
	zoneIPValue := new(big.Int).SetBytes(workerPrefix.Addr().AsSlice())

	for _, name := range zoneNames {
		zoneWorkerIP, _ := netip.AddrFromSlice(zoneIPValue.Bytes())
		zoneWorkerCidr := netip.PrefixFrom(zoneWorkerIP, workerPrefixLength)
		zoneIPValue.Add(zoneIPValue, delta)
		zones = append(zones, &gqlschema.AzureZoneInput{
			Name: name,
			Cidr: zoneWorkerCidr.String(),
		})
	}
	return zones
}
