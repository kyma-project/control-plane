package provider

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

type (
	FreemiumInput struct{}
)

func (i *FreemiumInput) Defaults() *gqlschema.ClusterConfigInput {
	// todo: maybe reuse the default with trial?
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
			MaxUnavailable: 1,
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

func (i *FreemiumInput) ApplyParameters(input *gqlschema.ClusterConfigInput, params internal.ProvisioningParameters) {

}

func (i *FreemiumInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileEvaluation
}
