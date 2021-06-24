package provider

import (
	"fmt"
	"math/rand"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const (
	DefaultOpenStackRegion = "eu-de-1"
)

type OpenStackInput struct {
}

func (p *OpenStackInput) Defaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       nil,
			MachineType:    "m2.xlarge",
			Region:         DefaultOpenStackRegion,
			Provider:       "openstack",
			WorkerCidr:     "10.250.0.0/19",
			AutoScalerMin:  2,
			AutoScalerMax:  4,
			MaxSurge:       4,
			MaxUnavailable: 0,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				OpenStackConfig: &gqlschema.OpenStackProviderConfigInput{
					Zones:                ZonesForOpenStack(DefaultOpenStackRegion),
					FloatingPoolName:     "FloatingIP-external-cp",
					CloudProfileName:     "converged-cloud-cp",
					LoadBalancerProvider: "f5",
				},
			},
		},
	}
}

func (p *OpenStackInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	if pp.Parameters.Region != nil && pp.Parameters.Zones == nil {
		input.GardenerConfig.ProviderSpecificConfig.OpenStackConfig.Zones = ZonesForOpenStack(*pp.Parameters.Region)
	}

	if len(pp.Parameters.Zones) > 0 {
		input.GardenerConfig.ProviderSpecificConfig.OpenStackConfig.Zones = pp.Parameters.Zones
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
