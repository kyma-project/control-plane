package model

import (
	"testing"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/testkit"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryRuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var purpose = gardener_types.ShootPurposeTesting

func Test_NewGardenerConfigFromJSON(t *testing.T) {

	gcpConfigJSON := `{"zones":["fix-gcp-zone-1", "fix-gcp-zone-2"]}`
	azureConfigJSON := `{"vnetCidr":"10.10.11.11/255", "zones":["fix-az-zone-1", "fix-az-zone-2"], "enableNatGateway":true, "idleConnectionTimeoutMinutes":4}`
	azureNoZonesConfigJSON := `{"vnetCidr":"10.10.11.11/255"}`
	azureZoneSubnetsConfigJSON := `{"vnetCidr":"10.10.11.11/255", "azureZones":[{"name":1,"cidr":"10.10.11.12/255"}, {"name":2,"cidr":"10.10.11.13/255"}], "enableNatGateway":true, "idleConnectionTimeoutMinutes":4}`
	awsConfigJSON := `{"vpcCidr":"10.10.11.11/255","awsZones":[{"name":"zone","publicCidr":"10.10.11.12/255","internalCidr":"10.10.11.13/255","workerCidr":"10.10.11.11/255"}]}
`
	singleZoneAwsConfigJSON := `{"zone":"zone","vpcCidr":"10.10.11.11/255","publicCidr":"10.10.11.12/255","internalCidr":"10.10.11.13/255"}`

	for _, testCase := range []struct {
		description                    string
		jsonData                       string
		expectedConfig                 GardenerProviderConfig
		expectedProviderSpecificConfig gqlschema.ProviderSpecificConfig
	}{
		{
			description: "should create GCP Gardener config",
			jsonData:    gcpConfigJSON,
			expectedConfig: &GCPGardenerConfig{
				ProviderSpecificConfig: ProviderSpecificConfig(gcpConfigJSON),
				input:                  &gqlschema.GCPProviderConfigInput{Zones: []string{"fix-gcp-zone-1", "fix-gcp-zone-2"}},
			},
			expectedProviderSpecificConfig: gqlschema.GCPProviderConfig{Zones: []string{"fix-gcp-zone-1", "fix-gcp-zone-2"}},
		},
		{
			description: "should create Azure Gardener config when zones passed",
			jsonData:    azureConfigJSON,
			expectedConfig: &AzureGardenerConfig{
				ProviderSpecificConfig: ProviderSpecificConfig(azureConfigJSON),
				input:                  &gqlschema.AzureProviderConfigInput{VnetCidr: "10.10.11.11/255", Zones: []string{"fix-az-zone-1", "fix-az-zone-2"}, EnableNatGateway: util.BoolPtr(true), IdleConnectionTimeoutMinutes: util.IntPtr(4)},
			},
			expectedProviderSpecificConfig: gqlschema.AzureProviderConfig{VnetCidr: util.StringPtr("10.10.11.11/255"), Zones: []string{"fix-az-zone-1", "fix-az-zone-2"}, EnableNatGateway: util.BoolPtr(true), IdleConnectionTimeoutMinutes: util.IntPtr(4)},
		},
		{
			description: "should create Azure Gardener config when no zones passed",
			jsonData:    azureNoZonesConfigJSON,
			expectedConfig: &AzureGardenerConfig{
				ProviderSpecificConfig: ProviderSpecificConfig(azureNoZonesConfigJSON),
				input:                  &gqlschema.AzureProviderConfigInput{VnetCidr: "10.10.11.11/255"},
			},
			expectedProviderSpecificConfig: gqlschema.AzureProviderConfig{VnetCidr: util.StringPtr("10.10.11.11/255")},
		},
		{
			description: "should create Azure Gardener config when subnets per zone input passed",
			jsonData:    azureZoneSubnetsConfigJSON,
			expectedConfig: &AzureGardenerConfig{
				ProviderSpecificConfig: ProviderSpecificConfig(azureZoneSubnetsConfigJSON),
				input: &gqlschema.AzureProviderConfigInput{
					VnetCidr: "10.10.11.11/255",
					AzureZones: []*gqlschema.AzureZoneInput{
						{
							Name: 1,
							Cidr: "10.10.11.12/255",
						},
						{
							Name: 2,
							Cidr: "10.10.11.13/255",
						},
					},
					EnableNatGateway:             util.BoolPtr(true),
					IdleConnectionTimeoutMinutes: util.IntPtr(4),
				},
			},
			expectedProviderSpecificConfig: gqlschema.AzureProviderConfig{
				VnetCidr: util.StringPtr("10.10.11.11/255"),
				AzureZones: []*gqlschema.AzureZone{
					{
						Name: 1,
						Cidr: "10.10.11.12/255",
					},
					{
						Name: 2,
						Cidr: "10.10.11.13/255",
					},
				},
				EnableNatGateway:             util.BoolPtr(true),
				IdleConnectionTimeoutMinutes: util.IntPtr(4)},
		},
		{
			description: "should create AWS Gardener config",
			jsonData:    awsConfigJSON,
			expectedConfig: &AWSGardenerConfig{
				ProviderSpecificConfig: ProviderSpecificConfig(awsConfigJSON),
				input: &gqlschema.AWSProviderConfigInput{
					AwsZones: []*gqlschema.AWSZoneInput{
						{
							Name:         "zone",
							PublicCidr:   "10.10.11.12/255",
							InternalCidr: "10.10.11.13/255",
							WorkerCidr:   "10.10.11.11/255",
						},
					},
					VpcCidr: "10.10.11.11/255",
				},
			},
			expectedProviderSpecificConfig: gqlschema.AWSProviderConfig{
				AwsZones: []*gqlschema.AWSZone{
					{
						Name:         util.StringPtr("zone"),
						PublicCidr:   util.StringPtr("10.10.11.12/255"),
						InternalCidr: util.StringPtr("10.10.11.13/255"),
						WorkerCidr:   util.StringPtr("10.10.11.11/255"),
					},
				},
				VpcCidr: util.StringPtr("10.10.11.11/255"),
			},
		},
		{
			description: "should create AWS Gardener config with single zone from old schema format",
			jsonData:    singleZoneAwsConfigJSON,
			expectedConfig: &AWSGardenerConfig{
				ProviderSpecificConfig: ProviderSpecificConfig(awsConfigJSON),
				input: &gqlschema.AWSProviderConfigInput{
					AwsZones: []*gqlschema.AWSZoneInput{
						{
							Name:         "zone",
							PublicCidr:   "10.10.11.12/255",
							InternalCidr: "10.10.11.13/255",
							WorkerCidr:   "10.10.11.11/255",
						},
					},
					VpcCidr: "10.10.11.11/255",
				},
			},
			expectedProviderSpecificConfig: gqlschema.AWSProviderConfig{
				AwsZones: []*gqlschema.AWSZone{
					{
						Name:         util.StringPtr("zone"),
						PublicCidr:   util.StringPtr("10.10.11.12/255"),
						InternalCidr: util.StringPtr("10.10.11.13/255"),
						WorkerCidr:   util.StringPtr("10.10.11.11/255"),
					},
				},
				VpcCidr: util.StringPtr("10.10.11.11/255"),
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// when
			gardenerProviderConfig, err := NewGardenerProviderConfigFromJSON(testCase.jsonData)

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedConfig, gardenerProviderConfig)

			// when
			providerSpecificConfig := gardenerProviderConfig.AsProviderSpecificConfig()

			// then
			assert.Equal(t, testCase.expectedProviderSpecificConfig, providerSpecificConfig)
		})
	}
}

func TestGardenerConfig_ToShootTemplate(t *testing.T) {

	zones := []string{"fix-zone-1", "fix-zone-2"}

	gcpGardenerProvider, err := NewGCPGardenerConfig(fixGCPGardenerInput(zones))
	require.NoError(t, err)

	azureGardenerProvider, err := NewAzureGardenerConfig(fixAzureGardenerInput(zones, util.BoolPtr(true)))
	require.NoError(t, err)

	azureNoZonesGardenerProvider, err := NewAzureGardenerConfig(fixAzureGardenerInput(nil, util.BoolPtr(false)))
	require.NoError(t, err)

	azureZoneSubnetsGardenerProvider, err := NewAzureGardenerConfig(fixAzureZoneSubnetsInput())
	require.NoError(t, err)

	awsGardenerProvider, err := NewAWSGardenerConfig(fixAWSGardenerInput())
	require.NoError(t, err)

	for _, testCase := range []struct {
		description           string
		provider              string
		providerConfig        GardenerProviderConfig
		expectedShootTemplate *gardener_types.Shoot
	}{
		{description: "should convert to Shoot template with GCP provider",
			provider:       "gcp",
			providerConfig: gcpGardenerProvider,
			expectedShootTemplate: &gardener_types.Shoot{
				ObjectMeta: v1.ObjectMeta{
					Name:      "cluster",
					Namespace: "gardener-namespace",
					Labels: map[string]string{
						"account":    "account",
						"subaccount": "sub-account",
					},
					Annotations: map[string]string{},
				},
				Spec: gardener_types.ShootSpec{
					CloudProfileName: "gcp",
					Networking: gardener_types.Networking{
						Type:  "calico",
						Nodes: util.StringPtr("10.10.10.10/255"),
					},
					SeedName:          util.StringPtr("eu"),
					SecretBindingName: "gardener-secret",
					Region:            "eu",
					Provider: gardener_types.Provider{
						Type: "gcp",
						ControlPlaneConfig: &apimachineryRuntime.RawExtension{
							Raw: []byte(`{"kind":"ControlPlaneConfig","apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","zone":"fix-zone-1"}`),
						},
						InfrastructureConfig: &apimachineryRuntime.RawExtension{
							Raw: []byte(`{"kind":"InfrastructureConfig","apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","networks":{"worker":"10.10.10.10/255","workers":"10.10.10.10/255"}}`),
						},
						Workers: []gardener_types.Worker{
							fixWorker([]string{"fix-zone-1", "fix-zone-2"}),
						},
					},
					Purpose:           &purpose,
					ExposureClassName: util.StringPtr("internet"),
					Kubernetes: gardener_types.Kubernetes{
						AllowPrivilegedContainers: util.BoolPtr(false),
						Version:                   "1.15",
						KubeAPIServer: &gardener_types.KubeAPIServerConfig{
							EnableBasicAuthentication: util.BoolPtr(false),
							OIDCConfig:                gardenerOidcConfig(oidcConfig()),
						},
					},
					Maintenance: &gardener_types.Maintenance{
						AutoUpdate: &gardener_types.MaintenanceAutoUpdate{
							KubernetesVersion:   true,
							MachineImageVersion: false,
						},
					},
					DNS: gardenerDnsConfig(dnsConfig()),
					Extensions: []gardener_types.Extension{
						{
							Type: "shoot-dns-service",
							ProviderConfig: &apimachineryRuntime.RawExtension{
								Raw: []byte(`{"apiVersion":"service.dns.extensions.gardener.cloud/v1alpha1","dnsProviderReplication":{"enabled":true},"kind":"DNSConfig"}`),
							},
						},
						{
							Type: "shoot-cert-service",
							ProviderConfig: &apimachineryRuntime.RawExtension{
								Raw: []byte(`{"apiVersion":"service.cert.extensions.gardener.cloud/v1alpha1","shootIssuers":{"enabled":true},"kind":"CertConfig"}`),
							},
						},
						{
							Type:     ShootNetworkingFilterExtensionType,
							Disabled: util.BoolPtr(true),
						},
					},
				},
			},
		},
		{description: "should convert to Shoot template with Azure provider when zones passed",
			provider:       "az",
			providerConfig: azureGardenerProvider,
			expectedShootTemplate: &gardener_types.Shoot{
				ObjectMeta: v1.ObjectMeta{
					Name:      "cluster",
					Namespace: "gardener-namespace",
					Labels: map[string]string{
						"account":    "account",
						"subaccount": "sub-account",
					},
					Annotations: map[string]string{},
				},
				Spec: gardener_types.ShootSpec{
					CloudProfileName: "az",
					Networking: gardener_types.Networking{
						Type:  "calico",
						Nodes: util.StringPtr("10.10.11.11/255"),
					},
					SeedName:          util.StringPtr("eu"),
					SecretBindingName: "gardener-secret",
					Region:            "eu",
					Provider: gardener_types.Provider{
						Type: "azure",
						ControlPlaneConfig: &apimachineryRuntime.RawExtension{
							Raw: []byte(`{"kind":"ControlPlaneConfig","apiVersion":"azure.provider.extensions.gardener.cloud/v1alpha1"}`),
						},
						InfrastructureConfig: &apimachineryRuntime.RawExtension{
							Raw: []byte(`{"kind":"InfrastructureConfig","apiVersion":"azure.provider.extensions.gardener.cloud/v1alpha1","networks":{"vnet":{"cidr":"10.10.11.11/255"},"workers":"10.10.10.10/255","natGateway":{"enabled":true,"idleConnectionTimeoutMinutes":4}},"zoned":true}`),
						},
						Workers: []gardener_types.Worker{
							fixWorker([]string{"fix-zone-1", "fix-zone-2"}),
						},
					},
					Purpose:           &purpose,
					ExposureClassName: util.StringPtr("internet"),
					Kubernetes: gardener_types.Kubernetes{
						AllowPrivilegedContainers: util.BoolPtr(false),
						Version:                   "1.15",
						KubeAPIServer: &gardener_types.KubeAPIServerConfig{
							EnableBasicAuthentication: util.BoolPtr(false),
							OIDCConfig:                gardenerOidcConfig(oidcConfig()),
						},
					},
					Maintenance: &gardener_types.Maintenance{
						AutoUpdate: &gardener_types.MaintenanceAutoUpdate{
							KubernetesVersion:   true,
							MachineImageVersion: false,
						},
					},
					DNS: gardenerDnsConfig(dnsConfig()),
					Extensions: []gardener_types.Extension{
						{
							Type: "shoot-dns-service",
							ProviderConfig: &apimachineryRuntime.RawExtension{
								Raw: []byte(`{"apiVersion":"service.dns.extensions.gardener.cloud/v1alpha1","dnsProviderReplication":{"enabled":true},"kind":"DNSConfig"}`),
							},
						},
						{
							Type: "shoot-cert-service",
							ProviderConfig: &apimachineryRuntime.RawExtension{
								Raw: []byte(`{"apiVersion":"service.cert.extensions.gardener.cloud/v1alpha1","shootIssuers":{"enabled":true},"kind":"CertConfig"}`),
							},
						},
						{
							Type:     ShootNetworkingFilterExtensionType,
							Disabled: util.BoolPtr(true),
						},
					},
				},
			},
		},
		{description: "should convert to Shoot template with Azure provider with no zones passed",
			provider:       "az",
			providerConfig: azureNoZonesGardenerProvider,
			expectedShootTemplate: &gardener_types.Shoot{
				ObjectMeta: v1.ObjectMeta{
					Name:      "cluster",
					Namespace: "gardener-namespace",
					Labels: map[string]string{
						"account":    "account",
						"subaccount": "sub-account",
					},
					Annotations: map[string]string{},
				},
				Spec: gardener_types.ShootSpec{
					CloudProfileName: "az",
					Networking: gardener_types.Networking{
						Type:  "calico",
						Nodes: util.StringPtr("10.10.11.11/255"),
					},
					SeedName:          util.StringPtr("eu"),
					SecretBindingName: "gardener-secret",
					Region:            "eu",
					Provider: gardener_types.Provider{
						Type: "azure",
						ControlPlaneConfig: &apimachineryRuntime.RawExtension{
							Raw: []byte(`{"kind":"ControlPlaneConfig","apiVersion":"azure.provider.extensions.gardener.cloud/v1alpha1"}`),
						},
						InfrastructureConfig: &apimachineryRuntime.RawExtension{
							Raw: []byte(`{"kind":"InfrastructureConfig","apiVersion":"azure.provider.extensions.gardener.cloud/v1alpha1","networks":{"vnet":{"cidr":"10.10.11.11/255"},"workers":"10.10.10.10/255"},"zoned":false}`),
						},
						Workers: []gardener_types.Worker{
							fixWorker(nil),
						},
					},
					Purpose:           &purpose,
					ExposureClassName: util.StringPtr("internet"),
					Kubernetes: gardener_types.Kubernetes{
						AllowPrivilegedContainers: util.BoolPtr(false),
						Version:                   "1.15",
						KubeAPIServer: &gardener_types.KubeAPIServerConfig{
							EnableBasicAuthentication: util.BoolPtr(false),
							OIDCConfig:                gardenerOidcConfig(oidcConfig()),
						},
					},
					Maintenance: &gardener_types.Maintenance{
						AutoUpdate: &gardener_types.MaintenanceAutoUpdate{
							KubernetesVersion:   true,
							MachineImageVersion: false,
						},
					},
					DNS: gardenerDnsConfig(dnsConfig()),
					Extensions: []gardener_types.Extension{
						{
							Type: "shoot-dns-service",
							ProviderConfig: &apimachineryRuntime.RawExtension{
								Raw: []byte(`{"apiVersion":"service.dns.extensions.gardener.cloud/v1alpha1","dnsProviderReplication":{"enabled":true},"kind":"DNSConfig"}`),
							},
						},
						{
							Type: "shoot-cert-service",
							ProviderConfig: &apimachineryRuntime.RawExtension{
								Raw: []byte(`{"apiVersion":"service.cert.extensions.gardener.cloud/v1alpha1","shootIssuers":{"enabled":true},"kind":"CertConfig"}`),
							},
						},
						{
							Type:     ShootNetworkingFilterExtensionType,
							Disabled: util.BoolPtr(true),
						},
					},
				},
			},
		},
		{description: "should convert to Shoot template with Azure provider when subnets per zone passed",
			provider:       "az",
			providerConfig: azureZoneSubnetsGardenerProvider,
			expectedShootTemplate: &gardener_types.Shoot{
				ObjectMeta: v1.ObjectMeta{
					Name:      "cluster",
					Namespace: "gardener-namespace",
					Labels: map[string]string{
						"account":    "account",
						"subaccount": "sub-account",
					},
					Annotations: map[string]string{},
				},
				Spec: gardener_types.ShootSpec{
					CloudProfileName: "az",
					Networking: gardener_types.Networking{
						Type:  "calico",
						Nodes: util.StringPtr("10.10.11.11/255"),
					},
					SeedName:          util.StringPtr("eu"),
					SecretBindingName: "gardener-secret",
					Region:            "eu",
					Provider: gardener_types.Provider{
						Type: "azure",
						ControlPlaneConfig: &apimachineryRuntime.RawExtension{
							Raw: []byte(`{"kind":"ControlPlaneConfig","apiVersion":"azure.provider.extensions.gardener.cloud/v1alpha1"}`),
						},
						InfrastructureConfig: &apimachineryRuntime.RawExtension{
							Raw: []byte(`{"kind":"InfrastructureConfig","apiVersion":"azure.provider.extensions.gardener.cloud/v1alpha1","networks":{"vnet":{"cidr":"10.10.11.11/255"},"zones":[{"name":1,"cidr":"10.10.11.12/255","natGateway":{"enabled":true,"idleConnectionTimeoutMinutes":4}},{"name":2,"cidr":"10.10.11.13/255","natGateway":{"enabled":true,"idleConnectionTimeoutMinutes":4}}]},"zoned":true}`),
						},
						Workers: []gardener_types.Worker{
							fixWorker([]string{"1", "2"}),
						},
					},
					Purpose:           &purpose,
					ExposureClassName: util.StringPtr("internet"),
					Kubernetes: gardener_types.Kubernetes{
						AllowPrivilegedContainers: util.BoolPtr(false),
						Version:                   "1.15",
						KubeAPIServer: &gardener_types.KubeAPIServerConfig{
							EnableBasicAuthentication: util.BoolPtr(false),
							OIDCConfig:                gardenerOidcConfig(oidcConfig()),
						},
					},
					Maintenance: &gardener_types.Maintenance{
						AutoUpdate: &gardener_types.MaintenanceAutoUpdate{
							KubernetesVersion:   true,
							MachineImageVersion: false,
						},
					},
					DNS: gardenerDnsConfig(dnsConfig()),
					Extensions: []gardener_types.Extension{
						{
							Type: "shoot-dns-service",
							ProviderConfig: &apimachineryRuntime.RawExtension{
								Raw: []byte(`{"apiVersion":"service.dns.extensions.gardener.cloud/v1alpha1","dnsProviderReplication":{"enabled":true},"kind":"DNSConfig"}`),
							},
						},
						{
							Type: "shoot-cert-service",
							ProviderConfig: &apimachineryRuntime.RawExtension{
								Raw: []byte(`{"apiVersion":"service.cert.extensions.gardener.cloud/v1alpha1","shootIssuers":{"enabled":true},"kind":"CertConfig"}`),
							},
						},
						{
							Type:     ShootNetworkingFilterExtensionType,
							Disabled: util.BoolPtr(true),
						},
					},
				},
			},
		},
		{description: "should convert to Shoot template with AWS provider",
			provider:       "aws",
			providerConfig: awsGardenerProvider,
			expectedShootTemplate: &gardener_types.Shoot{
				ObjectMeta: v1.ObjectMeta{
					Name:      "cluster",
					Namespace: "gardener-namespace",
					Labels: map[string]string{
						"account":    "account",
						"subaccount": "sub-account",
					},
					Annotations: map[string]string{},
				},
				Spec: gardener_types.ShootSpec{
					CloudProfileName: "aws",
					Networking: gardener_types.Networking{
						Type:  "calico",
						Nodes: util.StringPtr("10.10.11.11/255"),
					},
					SeedName:          util.StringPtr("eu"),
					SecretBindingName: "gardener-secret",
					Region:            "eu",
					Provider: gardener_types.Provider{
						Type: "aws",
						ControlPlaneConfig: &apimachineryRuntime.RawExtension{
							Raw: []byte(`{"kind":"ControlPlaneConfig","apiVersion":"aws.provider.extensions.gardener.cloud/v1alpha1"}`),
						},
						InfrastructureConfig: &apimachineryRuntime.RawExtension{
							Raw: []byte(`{"kind":"InfrastructureConfig","apiVersion":"aws.provider.extensions.gardener.cloud/v1alpha1","networks":{"vpc":{"cidr":"10.10.11.11/255"},"zones":[{"name":"zone","internal":"10.10.11.13/255","public":"10.10.11.12/255","workers":"10.10.11.12/255"}]}}`),
						},
						Workers: []gardener_types.Worker{
							fixWorker([]string{"zone"}),
						},
					},
					Purpose:           &purpose,
					ExposureClassName: util.StringPtr("internet"),
					Kubernetes: gardener_types.Kubernetes{
						AllowPrivilegedContainers: util.BoolPtr(false),
						Version:                   "1.15",
						KubeAPIServer: &gardener_types.KubeAPIServerConfig{
							EnableBasicAuthentication: util.BoolPtr(false),
							OIDCConfig:                gardenerOidcConfig(oidcConfig()),
						},
					},
					Maintenance: &gardener_types.Maintenance{
						AutoUpdate: &gardener_types.MaintenanceAutoUpdate{
							KubernetesVersion:   true,
							MachineImageVersion: false,
						},
					},
					DNS: gardenerDnsConfig(dnsConfig()),
					Extensions: []gardener_types.Extension{
						{
							Type: "shoot-dns-service",
							ProviderConfig: &apimachineryRuntime.RawExtension{
								Raw: []byte(`{"apiVersion":"service.dns.extensions.gardener.cloud/v1alpha1","dnsProviderReplication":{"enabled":true},"kind":"DNSConfig"}`),
							},
						},
						{
							Type: "shoot-cert-service",
							ProviderConfig: &apimachineryRuntime.RawExtension{
								Raw: []byte(`{"apiVersion":"service.cert.extensions.gardener.cloud/v1alpha1","shootIssuers":{"enabled":true},"kind":"CertConfig"}`),
							},
						},
						{
							Type:     ShootNetworkingFilterExtensionType,
							Disabled: util.BoolPtr(true),
						},
					},
				},
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerProviderConfig := fixGardenerConfig(testCase.provider, testCase.providerConfig)

			// when
			template, err := gardenerProviderConfig.ToShootTemplate("gardener-namespace", "account", "sub-account", oidcConfig(), dnsConfig())

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedShootTemplate, template)
		})
	}

}

func TestEditShootConfig(t *testing.T) {
	zones := []string{"fix-zone-1", "fix-zone-2"}

	initialShoot := testkit.NewTestShoot("shoot").
		WithAutoUpdate(false, false).
		WithWorkers(testkit.NewTestWorker("peon").ToWorker()).
		ToShoot()

	expectedShoot := testkit.NewTestShoot("shoot").
		WithKubernetesVersion("1.15").
		WithAutoUpdate(true, false).
		WithPurpose("testing").
		WithExposureClassName("internet").
		WithWorkers(
			testkit.NewTestWorker("peon").
				WithMachineType("machine").
				WithMachineImageAndVersion("gardenlinux", "25.0.0").
				WithVolume("SSD", 30).
				WithMinMax(1, 3).
				WithMaxSurge(30).
				WithMaxUnavailable(1).
				WithZones("fix-zone-1", "fix-zone-2").
				ToWorker()).
		ToShoot()

	awsProviderConfig, err := NewAWSGardenerConfig(fixAWSGardenerInput())
	require.NoError(t, err)

	azureProviderConfig, err := NewAzureGardenerConfig(fixAzureGardenerInput(zones, nil))
	require.NoError(t, err)

	gcpProviderConfig, err := NewGCPGardenerConfig(fixGCPGardenerInput(zones))
	require.NoError(t, err)

	for _, testCase := range []struct {
		description   string
		provider      string
		upgradeConfig GardenerConfig
		initialShoot  *gardener_types.Shoot
		expectedShoot *gardener_types.Shoot
	}{
		{description: "should edit AWS shoot template",
			provider:      "aws",
			upgradeConfig: fixGardenerConfig("aws", awsProviderConfig),
			initialShoot:  initialShoot.DeepCopy(),
			expectedShoot: func(s *gardener_types.Shoot) *gardener_types.Shoot {
				shoot := s.DeepCopy()
				shoot.Spec.Provider.Workers[0].Zones = []string{"zone"}
				return shoot
			}(expectedShoot), // fix of zones for AWS
		},
		{description: "should edit Azure shoot template",
			provider:      "az",
			upgradeConfig: fixGardenerConfig("az", azureProviderConfig),
			initialShoot:  initialShoot.DeepCopy(),
			expectedShoot: expectedShoot.DeepCopy(),
		},
		{description: "should edit GCP shoot template",
			provider:      "gcp",
			upgradeConfig: fixGardenerConfig("gcp", gcpProviderConfig),
			initialShoot:  initialShoot.DeepCopy(),
			expectedShoot: expectedShoot.DeepCopy(),
		},
		{description: "should update shoot networking extension",
			provider: "gcp",
			upgradeConfig: func(config GardenerConfig) GardenerConfig {
				config.ShootNetworkingFilterDisabled = util.BoolPtr(true)
				return config
			}(fixGardenerConfig("gcp", gcpProviderConfig)),
			initialShoot: initialShoot.DeepCopy(),
			expectedShoot: func(s *gardener_types.Shoot) *gardener_types.Shoot {
				shoot := s.DeepCopy()
				shoot.Spec.Extensions = append(shoot.Spec.Extensions, gardener_types.Extension{
					Type:     ShootNetworkingFilterExtensionType,
					Disabled: util.BoolPtr(true),
				})
				return shoot
			}(expectedShoot),
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerProviderConfig := testCase.upgradeConfig.GardenerProviderConfig

			// when
			err := gardenerProviderConfig.EditShootConfig(testCase.upgradeConfig, testCase.initialShoot)

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedShoot, testCase.initialShoot)
		})
	}

	for _, testCase := range []struct {
		description   string
		upgradeConfig GardenerConfig
		initialShoot  *gardener_types.Shoot
	}{
		{description: "should return error when no worker groups are assigned to shoot",
			upgradeConfig: fixGardenerConfig("az", azureProviderConfig),
			initialShoot:  testkit.NewTestShoot("shoot").ToShoot(),
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerProviderConfig := testCase.upgradeConfig.GardenerProviderConfig

			// when
			err := gardenerProviderConfig.EditShootConfig(testCase.upgradeConfig, testCase.initialShoot)

			// then
			require.Error(t, err)
		})
	}
}

func fixGardenerConfig(provider string, providerCfg GardenerProviderConfig) GardenerConfig {
	return GardenerConfig{
		ID:                                  "",
		ClusterID:                           "",
		Name:                                "cluster",
		ProjectName:                         "project",
		KubernetesVersion:                   "1.15",
		VolumeSizeGB:                        util.IntPtr(30),
		DiskType:                            util.StringPtr("SSD"),
		MachineType:                         "machine",
		MachineImage:                        util.StringPtr("gardenlinux"),
		MachineImageVersion:                 util.StringPtr("25.0.0"),
		Provider:                            provider,
		Purpose:                             util.StringPtr("testing"),
		LicenceType:                         nil,
		Seed:                                "eu",
		TargetSecret:                        "gardener-secret",
		Region:                              "eu",
		WorkerCidr:                          "10.10.10.10/255",
		AutoScalerMin:                       1,
		AutoScalerMax:                       3,
		MaxSurge:                            30,
		MaxUnavailable:                      1,
		EnableKubernetesVersionAutoUpdate:   true,
		EnableMachineImageVersionAutoUpdate: false,
		AllowPrivilegedContainers:           false,
		GardenerProviderConfig:              providerCfg,
		OIDCConfig:                          oidcConfig(),
		ExposureClassName:                   util.StringPtr("internet"),
		ShootNetworkingFilterDisabled:       nil,
	}
}

func fixAWSGardenerInput() *gqlschema.AWSProviderConfigInput {
	return &gqlschema.AWSProviderConfigInput{
		AwsZones: []*gqlschema.AWSZoneInput{
			{
				Name:         "zone",
				PublicCidr:   "10.10.11.12/255",
				InternalCidr: "10.10.11.13/255",
				WorkerCidr:   "10.10.11.12/255",
			},
		},
		VpcCidr: "10.10.11.11/255",
	}
}

func fixGCPGardenerInput(zones []string) *gqlschema.GCPProviderConfigInput {
	return &gqlschema.GCPProviderConfigInput{Zones: zones}
}

func fixAzureGardenerInput(zones []string, enableNAT *bool) *gqlschema.AzureProviderConfigInput {
	return &gqlschema.AzureProviderConfigInput{VnetCidr: "10.10.11.11/255", Zones: zones, EnableNatGateway: enableNAT, IdleConnectionTimeoutMinutes: util.IntPtr(4)}
}

func fixAzureZoneSubnetsInput() *gqlschema.AzureProviderConfigInput {
	return &gqlschema.AzureProviderConfigInput{
		VnetCidr:                     "10.10.11.11/255",
		EnableNatGateway:             util.BoolPtr(true),
		IdleConnectionTimeoutMinutes: util.IntPtr(4),
		AzureZones: []*gqlschema.AzureZoneInput{
			{
				Name: 1,
				Cidr: "10.10.11.12/255",
			},
			{
				Name: 2,
				Cidr: "10.10.11.13/255",
			},
		},
	}
}

func fixWorker(zones []string) gardener_types.Worker {
	return gardener_types.Worker{
		Name:           "cpu-worker-0",
		MaxSurge:       util.IntOrStringPtr(intstr.FromInt(30)),
		MaxUnavailable: util.IntOrStringPtr(intstr.FromInt(1)),
		Machine: gardener_types.Machine{
			Type: "machine",
			Image: &gardener_types.ShootMachineImage{
				Name:    "gardenlinux",
				Version: util.StringPtr("25.0.0"),
			},
		},
		Volume: &gardener_types.Volume{
			Type:       util.StringPtr("SSD"),
			VolumeSize: "30Gi",
		},
		Maximum: 3,
		Minimum: 1,
		Zones:   zones,
	}
}

func oidcConfig() *OIDCConfig {
	return &OIDCConfig{
		ClientID:       "9bd05ed7-a930-44e6-8c79-e6defeb1111",
		GroupsClaim:    "groups",
		IssuerURL:      "https://kymatest.accounts400.ondemand.com",
		SigningAlgs:    []string{"RS256"},
		UsernameClaim:  "sub",
		UsernamePrefix: "-",
	}
}

func dnsConfig() *DNSConfig {
	return &DNSConfig{
		Domain: "cluster.devtest.kyma.ondemand.com",
		Providers: []*DNSProvider{
			{
				DomainsInclude: []string{"devtest.kyma.ondemand.com"},
				Primary:        true,
				SecretName:     "aws_dns_domain_secrets_test_ingardenerconfig",
				Type:           "route53_type_test",
			},
		},
	}
}
