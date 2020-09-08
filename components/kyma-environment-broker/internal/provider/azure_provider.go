package provider

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const (
	DefaultAzureRegion = "westeurope"
)

var europeAzure = "westeurope"
var usAzure = "eastus"

var toAzureSpecific = map[string]*string{string(broker.Europe): &europeAzure, string(broker.Us): &usAzure}

type (
	AzureInput      struct{}
	AzureLiteInput  struct{}
	AzureTrialInput struct{}
)

func (p *AzureInput) Defaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       "Standard_LRS",
			VolumeSizeGb:   50,
			MachineType:    "Standard_D8_v3",
			Region:         DefaultAzureRegion,
			Provider:       "azure",
			WorkerCidr:     "10.250.0.0/19",
			AutoScalerMin:  3,
			AutoScalerMax:  10,
			MaxSurge:       4,
			MaxUnavailable: 1,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AzureConfig: &gqlschema.AzureProviderConfigInput{
					VnetCidr: "10.250.0.0/19",
				},
			},
		},
	}
}

func (p *AzureInput) ApplyParameters(input *gqlschema.ClusterConfigInput, params internal.ProvisioningParametersDTO) {
	updateSlice(&input.GardenerConfig.ProviderSpecificConfig.AzureConfig.Zones, params.Zones)
}

func (p *AzureLiteInput) Defaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       "Standard_LRS",
			VolumeSizeGb:   50,
			MachineType:    "Standard_D4_v3",
			Region:         DefaultAzureRegion,
			Provider:       "azure",
			WorkerCidr:     "10.250.0.0/19",
			AutoScalerMin:  3,
			AutoScalerMax:  4,
			MaxSurge:       4,
			MaxUnavailable: 1,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AzureConfig: &gqlschema.AzureProviderConfigInput{
					VnetCidr: "10.250.0.0/19",
				},
			},
		},
	}
}

func (p *AzureLiteInput) ApplyParameters(input *gqlschema.ClusterConfigInput, params internal.ProvisioningParametersDTO) {
	updateSlice(&input.GardenerConfig.ProviderSpecificConfig.AzureConfig.Zones, params.Zones)
}

func (p *AzureTrialInput) Defaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       "Standard_LRS",
			VolumeSizeGb:   50,
			MachineType:    "Standard_D4_v3",
			Region:         DefaultAzureRegion,
			Provider:       "azure",
			WorkerCidr:     "10.250.0.0/19",
			AutoScalerMin:  2,
			AutoScalerMax:  2,
			MaxSurge:       1,
			MaxUnavailable: 1,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AzureConfig: &gqlschema.AzureProviderConfigInput{
					VnetCidr: "10.250.0.0/19",
				},
			},
		},
	}
}

func (p *AzureTrialInput) ApplyParameters(input *gqlschema.ClusterConfigInput, params internal.ProvisioningParametersDTO) {
	if params.Region != nil {
		updateString(&input.GardenerConfig.Region, toAzureSpecific[*params.Region])
	}

	updateSlice(&input.GardenerConfig.ProviderSpecificConfig.AzureConfig.Zones, params.Zones)
}
