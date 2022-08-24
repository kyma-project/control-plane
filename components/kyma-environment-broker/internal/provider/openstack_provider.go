package provider

import (
	"fmt"
	"math/rand"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const (
	DefaultOpenStackRegion = "eu-de-2"
	DefaultExposureClass   = "converged-cloud-internet"
)

type OpenStackInput struct {
	FloatingPoolName string
}

func (p *OpenStackInput) Defaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:          nil,
			MachineType:       "g_c4_m16",
			Region:            DefaultOpenStackRegion,
			Provider:          "openstack",
			WorkerCidr:        "10.250.0.0/19",
			AutoScalerMin:     4,
			AutoScalerMax:     8,
			MaxSurge:          1,
			MaxUnavailable:    0,
			ExposureClassName: ptr.String(DefaultExposureClass),
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				OpenStackConfig: &gqlschema.OpenStackProviderConfigInput{
					Zones:                ZonesForOpenStack(DefaultOpenStackRegion),
					FloatingPoolName:     p.FloatingPoolName,
					CloudProfileName:     "converged-cloud-cp",
					LoadBalancerProvider: "f5",
				},
			},
		},
	}
}

func (p *OpenStackInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	if pp.Parameters.Region != nil && *pp.Parameters.Region != "" {
		input.GardenerConfig.ProviderSpecificConfig.OpenStackConfig.Zones = ZonesForOpenStack(*pp.Parameters.Region)
	}
}

func (p *OpenStackInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileProduction
}

func (p *OpenStackInput) Provider() internal.CloudProvider {
	return internal.Openstack
}

// openstackZones defines a possible suffixes for given OpenStack regions
// The table is tested in a unit test to check if all necessary regions are covered
var openstackZones = map[string]string{
	"eu-de-1": "abd",
	"eu-de-2": "abd",
	"ap-sa-1": "a",
}

func ZonesForOpenStack(region string) []string {
	zones, found := openstackZones[region]
	if !found {
		zones = "a"
	}
	zone := string(zones[rand.Intn(len(zones))])
	return []string{fmt.Sprintf("%s%s", region, zone)}
}
