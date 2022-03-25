package provider

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const (
	DefaultAzureRegion       = "eastus"
	DefaultAzureHAZonesCount = 3
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
	AzureInput      struct{}
	AzureLiteInput  struct{}
	AzureTrialInput struct {
		PlatformRegionMapping map[string]string
	}
	AzureFreemiumInput struct{}
	AzureHAInput       struct{}
)

func (p *AzureInput) Defaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       ptr.String("Standard_LRS"),
			VolumeSizeGb:   ptr.Integer(50),
			MachineType:    "Standard_D8_v3",
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
					Zones:    generateDefaultAzureZones(),
				},
			},
		},
	}
}

func (p *AzureInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	updateSlice(&input.GardenerConfig.ProviderSpecificConfig.AzureConfig.Zones, pp.Parameters.Zones)
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
					Zones:    generateDefaultAzureZones(),
				},
			},
		},
	}
}

func (p *AzureLiteInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	updateSlice(&input.GardenerConfig.ProviderSpecificConfig.AzureConfig.Zones, pp.Parameters.Zones)
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
					Zones:    generateDefaultAzureZones(),
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
		fmt.Println(*params.Region)
		updateString(&input.GardenerConfig.Region, toAzureSpecific[*params.Region])
		fmt.Println(input.GardenerConfig.Region)
		fmt.Println(toAzureSpecific[*params.Region])
	}

	updateSlice(&input.GardenerConfig.ProviderSpecificConfig.AzureConfig.Zones, params.Zones)
}

func (p *AzureTrialInput) Provider() internal.CloudProvider {
	return internal.Azure
}

func (p *AzureHAInput) Defaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       ptr.String("Standard_LRS"),
			VolumeSizeGb:   ptr.Integer(50),
			MachineType:    "Standard_D8_v3",
			Region:         DefaultAzureRegion,
			Provider:       "azure",
			WorkerCidr:     "10.250.0.0/19",
			AutoScalerMin:  1,
			AutoScalerMax:  10,
			MaxSurge:       1,
			MaxUnavailable: 0,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AzureConfig: &gqlschema.AzureProviderConfigInput{
					VnetCidr: "10.250.0.0/19",
					Zones:    generateMultipleAzureZones(DefaultAzureHAZonesCount),
				},
			},
		},
	}
}

func (p *AzureHAInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	if pp.Parameters.Zones == nil && pp.Parameters.ZonesCount != nil {
		updateSlice(&input.GardenerConfig.ProviderSpecificConfig.AzureConfig.Zones, generateMultipleAzureZones(*pp.Parameters.ZonesCount))
		return
	}
	updateSlice(&input.GardenerConfig.ProviderSpecificConfig.AzureConfig.Zones, pp.Parameters.Zones)
}

func (p *AzureHAInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileProduction
}

func (p *AzureHAInput) Provider() internal.CloudProvider {
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
