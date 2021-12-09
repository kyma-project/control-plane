package provider

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const (
	DefaultGCPRegion      = "europe-west3"
	DefaultGCPMachineType = "n2-standard-8"
)

var europeGcp = "europe-west3"
var usGcp = "us-central1"
var asiaGcp = "asia-south1"

var toGCPSpecific = map[string]*string{
	string(broker.Europe): &europeGcp,
	string(broker.Us):     &usGcp,
	string(broker.Asia):   &asiaGcp,
}

type (
	GcpInput      struct{}
	GcpTrialInput struct {
		PlatformRegionMapping map[string]string
	}
)

func (p *GcpInput) Defaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       ptr.String("pd-standard"),
			VolumeSizeGb:   ptr.Integer(50),
			MachineType:    DefaultGCPMachineType,
			Region:         DefaultGCPRegion,
			Provider:       "gcp",
			WorkerCidr:     "10.250.0.0/19",
			AutoScalerMin:  2,
			AutoScalerMax:  10,
			MaxSurge:       4,
			MaxUnavailable: 0,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				GcpConfig: &gqlschema.GCPProviderConfigInput{
					Zones: ZonesForGCPRegion(DefaultGCPRegion),
				},
			},
		},
	}
}

func (p *GcpInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	if pp.Parameters.Region != nil && pp.Parameters.Zones == nil {
		updateSlice(&input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, ZonesForGCPRegion(*pp.Parameters.Region))
	}

	updateSlice(&input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, pp.Parameters.Zones)
}

func (p *GcpInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileProduction
}

func (p *GcpInput) Provider() internal.CloudProvider {
	return internal.GCP
}

func (p *GcpTrialInput) Defaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       ptr.String("pd-standard"),
			VolumeSizeGb:   ptr.Integer(30),
			MachineType:    "n1-standard-4",
			Region:         DefaultGCPRegion,
			Provider:       "gcp",
			WorkerCidr:     "10.250.0.0/19",
			AutoScalerMin:  1,
			AutoScalerMax:  1,
			MaxSurge:       1,
			MaxUnavailable: 0,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				GcpConfig: &gqlschema.GCPProviderConfigInput{
					Zones: ZonesForGCPRegion(DefaultGCPRegion),
				},
			},
		},
	}
}

func (p *GcpTrialInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	params := pp.Parameters
	var region string

	// if there is a platform region - use it
	if pp.PlatformRegion != "" {
		abstractRegion, found := p.PlatformRegionMapping[pp.PlatformRegion]
		if found {
			region = *toGCPSpecific[abstractRegion]
		}
	}

	// if the user provides a region - use this one
	if params.Region != nil {
		region = *toGCPSpecific[*params.Region]
	}

	// region is not empty - it means override the default one
	var zones []string
	if region != "" {
		updateString(&input.GardenerConfig.Region, &region)
		updateSlice(&input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, ZonesForGCPRegion(region))
	}

	updateSlice(&input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, zones)
}

func (p *GcpTrialInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileEvaluation
}

func (p *GcpTrialInput) Provider() internal.CloudProvider {
	return internal.GCP
}

func ZonesForGCPRegion(region string) []string {
	var zones []string

	for _, name := range []string{"a"} {
		zones = append(zones, fmt.Sprintf("%s-%s", region, name))
	}

	return zones
}
