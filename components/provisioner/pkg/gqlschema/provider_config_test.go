package gqlschema

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGardenerConfig_UnmarshalJSON(t *testing.T) {

	azureProviderCfgNoZones := &AzureProviderConfig{VnetCidr: util.PtrTo("10.10.11.11/25")}
	azureProviderCfg := &AzureProviderConfig{VnetCidr: util.PtrTo("10.10.11.11/25"), Zones: []string{"az-zone-1", "az-zone-2"}}
	gcpProviderCfg := &GCPProviderConfig{Zones: []string{"gcp-zone-1", "gcp-zone-2"}}
	awsProviderCfg := &AWSProviderConfig{
		AwsZones: []*AWSZone{},
		VpcCidr:  util.PtrTo("10.10.10.11/25"),
	}
	openstackProviderCfg := &OpenStackProviderConfig{
		Zones:                []string{"eu-de-1a"},
		FloatingPoolName:     "FloatingIP-external-cp",
		CloudProfileName:     "converged-cloud-cp",
		LoadBalancerProvider: "f5",
	}

	for _, testCase := range []struct {
		description    string
		gardenerConfig GardenerConfig
	}{
		{
			description:    "gardener cluster with Azure with no zones passed",
			gardenerConfig: newGardenerClusterCfg(fixGardenerConfig("azure"), azureProviderCfgNoZones),
		},
		{
			description:    "gardener cluster with Azure",
			gardenerConfig: newGardenerClusterCfg(fixGardenerConfig("azure"), azureProviderCfg),
		},
		{
			description:    "gardener cluster with GCP",
			gardenerConfig: newGardenerClusterCfg(fixGardenerConfig("gcp"), gcpProviderCfg),
		},
		{
			description:    "gardener cluster with AWS",
			gardenerConfig: newGardenerClusterCfg(fixGardenerConfig("aws"), awsProviderCfg),
		},
		{
			description:    "gardener cluster with Openstack",
			gardenerConfig: newGardenerClusterCfg(fixGardenerConfig("openstack"), openstackProviderCfg),
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			marshalled, err := json.Marshal(testCase.gardenerConfig)
			require.NoError(t, err)

			var unmarshalledConfig GardenerConfig

			// when
			err = json.NewDecoder(bytes.NewBuffer(marshalled)).Decode(&unmarshalledConfig)
			require.NoError(t, err)

			// then
			assert.Equal(t, testCase.gardenerConfig, unmarshalledConfig)
		})
	}

}

func newGardenerClusterCfg(gardenerCfg GardenerConfig, providerCfg ProviderSpecificConfig) GardenerConfig {
	gardenerCfg.ProviderSpecificConfig = providerCfg

	return gardenerCfg
}

func fixGardenerConfig(providerName string) GardenerConfig {
	return GardenerConfig{
		Name:              util.PtrTo("name"),
		KubernetesVersion: util.PtrTo("1.16"),
		VolumeSizeGb:      util.PtrTo(50),
		MachineType:       util.PtrTo("machine"),
		Region:            util.PtrTo("region"),
		Provider:          util.PtrTo(providerName),
		Seed:              util.PtrTo("seed"),
		TargetSecret:      util.PtrTo("secret"),
		DiskType:          util.PtrTo("disk"),
		WorkerCidr:        util.PtrTo("10.10.10.10/25"),
		AutoScalerMin:     util.PtrTo(1),
		AutoScalerMax:     util.PtrTo(4),
		MaxSurge:          util.PtrTo(25),
		MaxUnavailable:    util.PtrTo(2),
	}
}
