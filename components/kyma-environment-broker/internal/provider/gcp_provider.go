package provider

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const (
	DefaultGCPRegion = "europe-west4"
)

var europeGcp = "europe-west4"
var usGcp = "us-east4"
var asiaGcp = "asia-southeast1"

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
			DiskType:       "pd-standard",
			VolumeSizeGb:   30,
			MachineType:    "n1-standard-4",
			Region:         DefaultGCPRegion,
			Provider:       "gcp",
			WorkerCidr:     "10.250.0.0/19",
			AutoScalerMin:  3,
			AutoScalerMax:  4,
			MaxSurge:       4,
			MaxUnavailable: 1,
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

func (p *GcpTrialInput) Defaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       "pd-standard",
			VolumeSizeGb:   30,
			MachineType:    "n1-standard-4",
			Region:         DefaultGCPRegion,
			Provider:       "gcp",
			WorkerCidr:     "10.250.0.0/19",
			AutoScalerMin:  1,
			AutoScalerMax:  1,
			MaxSurge:       1,
			MaxUnavailable: 1,
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

func ZonesForGCPRegion(region string) []string {
	var zones []string

	for _, name := range []string{"a"} {
		zones = append(zones, fmt.Sprintf("%s-%s", region, name))
	}

	return zones
}
