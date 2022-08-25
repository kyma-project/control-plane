package provider

import (
	"fmt"
	"math/rand"
	"strconv"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const (
	DefaultAzureRegion         = "eastus"
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
		MultiZone bool
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
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       ptr.String("Standard_LRS"),
			VolumeSizeGb:   ptr.Integer(50),
			MachineType:    "Standard_D4_v3",
			Region:         DefaultAzureRegion,
			Provider:       "azure",
			WorkerCidr:     "10.250.0.0/16",
			AutoScalerMin:  3,
			AutoScalerMax:  20,
			MaxSurge:       1,
			MaxUnavailable: 0,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AzureConfig: &gqlschema.AzureProviderConfigInput{
					VnetCidr:         "10.250.0.0/16",
					AzureZones:       generateMultipleAzureZones(generateRandomAzureZones(zonesCount)),
					EnableNatGateway: ptr.Bool(true),
				},
			},
		},
	}
}

func (p *AzureInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	// explicit zones list is provided
	if len(pp.Parameters.Zones) > 0 {
		zones := []int{}
		for _, inputZone := range pp.Parameters.Zones {
			zone, err := strconv.Atoi(inputZone)
			if err != nil || zone < 1 || zone > 3 {
				continue
			}
			zones = append(zones, zone)
		}
		input.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones = generateMultipleAzureZones(zones)
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
			WorkerCidr:     "10.250.0.0/19",
			AutoScalerMin:  2,
			AutoScalerMax:  10,
			MaxSurge:       1,
			MaxUnavailable: 0,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AzureConfig: &gqlschema.AzureProviderConfigInput{
					VnetCidr: "10.250.0.0/19",
					AzureZones: []*gqlschema.AzureZoneInput{
						{
							Name: generateRandomAzureZone(),
							Cidr: "10.250.0.0/19",
						},
					},
					EnableNatGateway: ptr.Bool(true),
				},
			},
		},
	}
}

func (p *AzureLiteInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
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
			WorkerCidr:     "10.250.0.0/19",
			AutoScalerMin:  1,
			AutoScalerMax:  1,
			MaxSurge:       1,
			MaxUnavailable: 0,
			Purpose:        &trialPurpose,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AzureConfig: &gqlschema.AzureProviderConfigInput{
					VnetCidr: "10.250.0.0/19",
					AzureZones: []*gqlschema.AzureZoneInput{
						{
							Name: generateRandomAzureZone(),
							Cidr: "10.250.0.0/19",
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

func generateMultipleAzureZones(zoneNames []int) []*gqlschema.AzureZoneInput {
	subnetFmt := "10.250.%d.0/19"
	zones := []*gqlschema.AzureZoneInput{}
	for i, zone := range zoneNames {
		zones = append(zones, &gqlschema.AzureZoneInput{
			Name: zone,
			Cidr: fmt.Sprintf(subnetFmt, i*32),
		})
	}

	return zones
}
