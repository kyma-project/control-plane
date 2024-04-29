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
var gardenerSecret = "gardener-secret"

func Test_NewGardenerConfigFromJSON(t *testing.T) {

	gcpConfigJSON := `{"zones":["fix-gcp-zone-1", "fix-gcp-zone-2"]}`
	azureConfigJSON := `{"vnetCidr":"10.10.11.11/255", "zones":["fix-az-zone-1", "fix-az-zone-2"], "enableNatGateway":true, "idleConnectionTimeoutMinutes":4}`
	azureNoZonesConfigJSON := `{"vnetCidr":"10.10.11.11/255"}`
	azureZoneSubnetsConfigJSON := `{"vnetCidr":"10.10.11.11/255", "azureZones":[{"name":1,"cidr":"10.10.11.12/255"}, {"name":2,"cidr":"10.10.11.13/255"}], "enableNatGateway":true, "idleConnectionTimeoutMinutes":4}`
	awsConfigJSON := `{"vpcCidr":"10.10.11.11/255","awsZones":[{"name":"zone","publicCidr":"10.10.11.12/255","internalCidr":"10.10.11.13/255","workerCidr":"10.10.11.11/255"}], "enableIMDSv2": true}
`

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
				input:                  &gqlschema.AzureProviderConfigInput{VnetCidr: "10.10.11.11/255", Zones: []string{"fix-az-zone-1", "fix-az-zone-2"}, EnableNatGateway: util.PtrTo(true), IdleConnectionTimeoutMinutes: util.PtrTo(4)},
			},
			expectedProviderSpecificConfig: gqlschema.AzureProviderConfig{VnetCidr: util.PtrTo("10.10.11.11/255"), Zones: []string{"fix-az-zone-1", "fix-az-zone-2"}, EnableNatGateway: util.PtrTo(true), IdleConnectionTimeoutMinutes: util.PtrTo(4)},
		},
		{
			description: "should create Azure Gardener config when no zones passed",
			jsonData:    azureNoZonesConfigJSON,
			expectedConfig: &AzureGardenerConfig{
				ProviderSpecificConfig: ProviderSpecificConfig(azureNoZonesConfigJSON),
				input:                  &gqlschema.AzureProviderConfigInput{VnetCidr: "10.10.11.11/255"},
			},
			expectedProviderSpecificConfig: gqlschema.AzureProviderConfig{VnetCidr: util.PtrTo("10.10.11.11/255")},
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
					EnableNatGateway:             util.PtrTo(true),
					IdleConnectionTimeoutMinutes: util.PtrTo(4),
				},
			},
			expectedProviderSpecificConfig: gqlschema.AzureProviderConfig{
				VnetCidr: util.PtrTo("10.10.11.11/255"),
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
				EnableNatGateway:             util.PtrTo(true),
				IdleConnectionTimeoutMinutes: util.PtrTo(4)},
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
					VpcCidr:      "10.10.11.11/255",
					EnableIMDSv2: util.PtrTo(true),
				},
			},
			expectedProviderSpecificConfig: gqlschema.AWSProviderConfig{
				AwsZones: []*gqlschema.AWSZone{
					{
						Name:         util.PtrTo("zone"),
						PublicCidr:   util.PtrTo("10.10.11.12/255"),
						InternalCidr: util.PtrTo("10.10.11.13/255"),
						WorkerCidr:   util.PtrTo("10.10.11.11/255"),
					},
				},
				VpcCidr:      util.PtrTo("10.10.11.11/255"),
				EnableIMDSv2: util.PtrTo(true),
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

	azureGardenerProvider, err := NewAzureGardenerConfig(fixAzureGardenerInput(zones, util.PtrTo(true)))
	require.NoError(t, err)

	azureNoZonesGardenerProvider, err := NewAzureGardenerConfig(fixAzureGardenerInput(nil, util.PtrTo(false)))
	require.NoError(t, err)

	azureZoneSubnetsGardenerProvider, err := NewAzureGardenerConfig(fixAzureZoneSubnetsInput(true))
	require.NoError(t, err)

	awsGardenerProvider, err := NewAWSGardenerConfig(fixAWSGardenerInput(true))
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
					Annotations: map[string]string{
						"support.gardener.cloud/eu-access-for-cluster-nodes": "true",
					},
				},
				Spec: gardener_types.ShootSpec{
					CloudProfileName: "gcp",
					Networking: &gardener_types.Networking{
						Type:     &networkingType,
						Nodes:    util.PtrTo("10.10.10.10/255"),
						Pods:     util.PtrTo("10.10.11.10/24"),
						Services: util.PtrTo("10.10.12.10/24"),
					},
					SeedName:          util.PtrTo("eu"),
					SecretBindingName: &gardenerSecret,
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
							fixWorker([]string{"fix-zone-1", "fix-zone-2"}, nil),
						},
					},
					Purpose:           &purpose,
					ExposureClassName: util.PtrTo("internet"),
					Kubernetes: gardener_types.Kubernetes{
						Version: "1.15",
						KubeAPIServer: &gardener_types.KubeAPIServerConfig{
							OIDCConfig: gardenerOidcConfig(oidcConfig()),
						},
						EnableStaticTokenKubeconfig: util.PtrTo(false),
					},
					Maintenance: &gardener_types.Maintenance{
						AutoUpdate: &gardener_types.MaintenanceAutoUpdate{
							KubernetesVersion:   true,
							MachineImageVersion: util.PtrTo(false),
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
							Disabled: util.PtrTo(true),
						},
					},
					ControlPlane: &gardener_types.ControlPlane{
						HighAvailability: &gardener_types.HighAvailability{
							FailureTolerance: gardener_types.FailureTolerance{
								Type: gardener_types.FailureToleranceTypeZone,
							},
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
					Annotations: map[string]string{
						"support.gardener.cloud/eu-access-for-cluster-nodes": "true",
					},
				},
				Spec: gardener_types.ShootSpec{
					CloudProfileName: "az",
					Networking: &gardener_types.Networking{
						Type:     &networkingType,
						Nodes:    util.PtrTo("10.10.11.11/255"),
						Pods:     util.PtrTo("10.10.11.10/24"),
						Services: util.PtrTo("10.10.12.10/24"),
					},
					SeedName:          util.PtrTo("eu"),
					SecretBindingName: &gardenerSecret,
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
							fixWorker([]string{"fix-zone-1", "fix-zone-2"}, nil),
						},
					},
					Purpose:           &purpose,
					ExposureClassName: util.PtrTo("internet"),
					Kubernetes: gardener_types.Kubernetes{
						Version: "1.15",
						KubeAPIServer: &gardener_types.KubeAPIServerConfig{
							OIDCConfig: gardenerOidcConfig(oidcConfig()),
						},
						EnableStaticTokenKubeconfig: util.PtrTo(false),
					},
					Maintenance: &gardener_types.Maintenance{
						AutoUpdate: &gardener_types.MaintenanceAutoUpdate{
							KubernetesVersion:   true,
							MachineImageVersion: util.PtrTo(false),
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
							Disabled: util.PtrTo(true),
						},
					},
					ControlPlane: &gardener_types.ControlPlane{
						HighAvailability: &gardener_types.HighAvailability{
							FailureTolerance: gardener_types.FailureTolerance{
								Type: gardener_types.FailureToleranceTypeZone,
							},
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
					Annotations: map[string]string{
						"support.gardener.cloud/eu-access-for-cluster-nodes": "true",
					},
				},
				Spec: gardener_types.ShootSpec{
					CloudProfileName: "az",
					Networking: &gardener_types.Networking{
						Type:     &networkingType,
						Nodes:    util.PtrTo("10.10.11.11/255"),
						Pods:     util.PtrTo("10.10.11.10/24"),
						Services: util.PtrTo("10.10.12.10/24"),
					},
					SeedName:          util.PtrTo("eu"),
					SecretBindingName: &gardenerSecret,
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
							fixWorker(nil, nil),
						},
					},
					Purpose:           &purpose,
					ExposureClassName: util.PtrTo("internet"),
					Kubernetes: gardener_types.Kubernetes{
						Version: "1.15",
						KubeAPIServer: &gardener_types.KubeAPIServerConfig{
							OIDCConfig: gardenerOidcConfig(oidcConfig()),
						},
						EnableStaticTokenKubeconfig: util.PtrTo(false),
					},
					Maintenance: &gardener_types.Maintenance{
						AutoUpdate: &gardener_types.MaintenanceAutoUpdate{
							KubernetesVersion:   true,
							MachineImageVersion: util.PtrTo(false),
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
							Disabled: util.PtrTo(true),
						},
					},
					ControlPlane: &gardener_types.ControlPlane{
						HighAvailability: &gardener_types.HighAvailability{
							FailureTolerance: gardener_types.FailureTolerance{
								Type: gardener_types.FailureToleranceTypeZone,
							},
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
					Annotations: map[string]string{
						"support.gardener.cloud/eu-access-for-cluster-nodes": "true",
					},
				},
				Spec: gardener_types.ShootSpec{
					CloudProfileName: "az",
					Networking: &gardener_types.Networking{
						Type:     &networkingType,
						Nodes:    util.PtrTo("10.10.11.11/255"),
						Pods:     util.PtrTo("10.10.11.10/24"),
						Services: util.PtrTo("10.10.12.10/24"),
					},
					SeedName:          util.PtrTo("eu"),
					SecretBindingName: &gardenerSecret,
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
							fixWorker([]string{"1", "2"}, nil),
						},
					},
					Purpose:           &purpose,
					ExposureClassName: util.PtrTo("internet"),
					Kubernetes: gardener_types.Kubernetes{
						Version: "1.15",
						KubeAPIServer: &gardener_types.KubeAPIServerConfig{
							OIDCConfig: gardenerOidcConfig(oidcConfig()),
						},
						EnableStaticTokenKubeconfig: util.PtrTo(false),
					},
					Maintenance: &gardener_types.Maintenance{
						AutoUpdate: &gardener_types.MaintenanceAutoUpdate{
							KubernetesVersion:   true,
							MachineImageVersion: util.PtrTo(false),
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
							Disabled: util.PtrTo(true),
						},
					},
					ControlPlane: &gardener_types.ControlPlane{
						HighAvailability: &gardener_types.HighAvailability{
							FailureTolerance: gardener_types.FailureTolerance{
								Type: gardener_types.FailureToleranceTypeZone,
							},
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
					Annotations: map[string]string{
						"support.gardener.cloud/eu-access-for-cluster-nodes": "true",
					},
				},
				Spec: gardener_types.ShootSpec{
					CloudProfileName: "aws",
					Networking: &gardener_types.Networking{
						Type:     &networkingType,
						Nodes:    util.PtrTo("10.10.11.11/255"),
						Pods:     util.PtrTo("10.10.11.10/24"),
						Services: util.PtrTo("10.10.12.10/24"),
					},
					SeedName:          util.PtrTo("eu"),
					SecretBindingName: &gardenerSecret,
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
							fixWorker([]string{"zone"}, &apimachineryRuntime.RawExtension{
								Raw: []byte(`{"kind":"WorkerConfig","apiVersion":"aws.provider.extensions.gardener.cloud/v1alpha1","instanceMetadataOptions":{"httpTokens":"required","httpPutResponseHopLimit":2}}`),
							}),
						},
					},
					Purpose:           &purpose,
					ExposureClassName: util.PtrTo("internet"),
					Kubernetes: gardener_types.Kubernetes{
						Version: "1.15",
						KubeAPIServer: &gardener_types.KubeAPIServerConfig{
							OIDCConfig: gardenerOidcConfig(oidcConfig()),
						},
						EnableStaticTokenKubeconfig: util.PtrTo(false),
					},
					Maintenance: &gardener_types.Maintenance{
						AutoUpdate: &gardener_types.MaintenanceAutoUpdate{
							KubernetesVersion:   true,
							MachineImageVersion: util.PtrTo(false),
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
							Disabled: util.PtrTo(true),
						},
					},
					ControlPlane: &gardener_types.ControlPlane{
						HighAvailability: &gardener_types.HighAvailability{
							FailureTolerance: gardener_types.FailureTolerance{
								Type: gardener_types.FailureToleranceTypeZone,
							},
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

func TestAdjustStaticKubeconfigFlagK8s126(t *testing.T) {
	//given old (1.26) shoot and request to upgrade not relevant to k8s version
	config := GardenerConfig{}
	shoot := testkit.NewTestShoot("shoot").WithKubernetesVersion("1.26.8")
	shoot.ToShoot().Spec.Kubernetes.EnableStaticTokenKubeconfig = util.PtrTo(true)

	//when
	adjustStaticKubeconfigFlag(config, shoot.ToShoot())

	//then
	assert.Equal(t, util.PtrTo(true), shoot.ToShoot().Spec.Kubernetes.EnableStaticTokenKubeconfig)
}

func TestAdjustStaticKubeconfigFlagForK8s127(t *testing.T) {
	//given old shoot config being 1.26 and upgrade requesting updating to 1.27 version
	config := GardenerConfig{
		KubernetesVersion: "1.27.8",
	}

	shoot := testkit.NewTestShoot("shoot").WithKubernetesVersion("1.26.8")
	shoot.ToShoot().Spec.Kubernetes.EnableStaticTokenKubeconfig = util.PtrTo(true)

	//when
	adjustStaticKubeconfigFlag(config, shoot.ToShoot())

	//then
	assert.Equal(t, util.PtrTo(false), shoot.ToShoot().Spec.Kubernetes.EnableStaticTokenKubeconfig)
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
				ToWorker()).
		ToShoot()

	awsProviderConfig, err := NewAWSGardenerConfig(fixAWSGardenerInput(false))
	require.NoError(t, err)

	awsProviderConfigWithEnableIMDSv2, err := NewAWSGardenerConfig(fixAWSGardenerInput(true))
	require.NoError(t, err)

	azureProviderConfig, err := NewAzureGardenerConfig(fixAzureGardenerInput(zones, nil))
	require.NoError(t, err)

	azureProviderConfigEnableNAT, err := NewAzureGardenerConfig(fixAzureGardenerInput([]string{}, util.PtrTo(true)))
	require.NoError(t, err)

	azureProviderConfigWithZonesAndNATEnabled, err := NewAzureGardenerConfig(fixAzureZoneSubnetsInput(true))
	require.NoError(t, err)

	gcpProviderConfig, err := NewGCPGardenerConfig(fixGCPGardenerInput(zones))
	require.NoError(t, err)

	initialShootWithInfrastructureConfig := initialShoot.DeepCopy()
	initialShootWithInfrastructureConfig.Spec.Provider.InfrastructureConfig = &apimachineryRuntime.RawExtension{
		Raw: []byte(azureProviderConfig.RawJSON()),
	}

	expectedShootConfigWithIMDSv2Enabled := expectedShoot.DeepCopy()
	expectedShootConfigWithIMDSv2Enabled.Spec.Provider.Workers[0].ProviderConfig = &apimachineryRuntime.RawExtension{
		Raw: []byte(`{"kind":"WorkerConfig","apiVersion":"aws.provider.extensions.gardener.cloud/v1alpha1","instanceMetadataOptions":{"httpTokens":"required","httpPutResponseHopLimit":2}}`),
	}

	expectedShootWithNATEnabled := expectedShoot.DeepCopy()
	expectedShootWithNATEnabled.Spec.Provider.InfrastructureConfig = &apimachineryRuntime.RawExtension{
		Raw: []byte(`{"networks":{"vnet":{"cidr":"10.10.11.11/255"},"natGateway":{"enabled":true,"idleConnectionTimeoutMinutes":4}},"zoned":false}`),
	}

	initialShootWithZones := initialShoot.DeepCopy()
	initialShootWithZones.Spec.Provider.InfrastructureConfig = &apimachineryRuntime.RawExtension{
		Raw: []byte(`{"kind":"InfrastructureConfig","apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","networks":{"zones":[{"name": 0}, {"name": 1}]}}`),
	}

	expectedShootWithZonesAndNATEnabled := expectedShoot.DeepCopy()
	expectedShootWithZonesAndNATEnabled.Spec.Provider.InfrastructureConfig = &apimachineryRuntime.RawExtension{
		Raw: []byte(`{"kind":"InfrastructureConfig","apiVersion":"gcp.provider.extensions.gardener.cloud/v1alpha1","networks":{"vnet":{"cidr":"10.10.11.11/255"},"zones":[{"name":0,"cidr":"","natGateway":{"enabled":true,"idleConnectionTimeoutMinutes":4}},{"name":1,"cidr":"","natGateway":{"enabled":true,"idleConnectionTimeoutMinutes":4}}]},"zoned":false}`),
	}

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
			expectedShoot: expectedShoot.DeepCopy(),
		},
		{description: "should edit AWS shoot template with IMDSv2 enabled",
			provider:      "aws",
			upgradeConfig: fixGardenerConfig("aws", awsProviderConfigWithEnableIMDSv2),
			initialShoot:  initialShoot.DeepCopy(),
			expectedShoot: expectedShootConfigWithIMDSv2Enabled.DeepCopy(),
		},
		{description: "should edit Azure shoot template",
			provider:      "az",
			upgradeConfig: fixGardenerConfig("az", azureProviderConfig),
			initialShoot:  initialShoot.DeepCopy(),
			expectedShoot: expectedShoot.DeepCopy(),
		},
		{description: "should edit Azure shoot template with NAT enabled",
			provider:      "az",
			upgradeConfig: fixGardenerConfig("az", azureProviderConfigEnableNAT),
			initialShoot:  initialShootWithInfrastructureConfig.DeepCopy(),
			expectedShoot: expectedShootWithNATEnabled.DeepCopy(),
		},
		{description: "should edit Azure shoot template with Azure Zones and NAT enabled",
			provider:      "az",
			upgradeConfig: fixGardenerConfig("az", azureProviderConfigWithZonesAndNATEnabled),
			initialShoot:  initialShootWithZones.DeepCopy(),
			expectedShoot: expectedShootWithZonesAndNATEnabled.DeepCopy(),
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
				config.ShootNetworkingFilterDisabled = util.PtrTo(true)
				return config
			}(fixGardenerConfig("gcp", gcpProviderConfig)),
			initialShoot: initialShoot.DeepCopy(),
			expectedShoot: func(s *gardener_types.Shoot) *gardener_types.Shoot {
				shoot := s.DeepCopy()
				shoot.Spec.Extensions = append(shoot.Spec.Extensions, gardener_types.Extension{
					Type:     ShootNetworkingFilterExtensionType,
					Disabled: util.PtrTo(true),
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
		VolumeSizeGB:                        util.PtrTo(30),
		DiskType:                            util.PtrTo("SSD"),
		MachineType:                         "machine",
		MachineImage:                        util.PtrTo("gardenlinux"),
		MachineImageVersion:                 util.PtrTo("25.0.0"),
		Provider:                            provider,
		Purpose:                             util.PtrTo("testing"),
		LicenceType:                         nil,
		Seed:                                "eu",
		TargetSecret:                        "gardener-secret",
		Region:                              "eu",
		WorkerCidr:                          "10.10.10.10/255",
		PodsCIDR:                            util.PtrTo("10.10.11.10/24"),
		ServicesCIDR:                        util.PtrTo("10.10.12.10/24"),
		AutoScalerMin:                       1,
		AutoScalerMax:                       3,
		MaxSurge:                            30,
		MaxUnavailable:                      1,
		EnableKubernetesVersionAutoUpdate:   true,
		EnableMachineImageVersionAutoUpdate: false,
		GardenerProviderConfig:              providerCfg,
		OIDCConfig:                          oidcConfig(),
		ExposureClassName:                   util.PtrTo("internet"),
		ShootNetworkingFilterDisabled:       nil,
		ControlPlaneFailureTolerance:        util.PtrTo("zone"),
		EuAccess:                            true,
	}
}

func fixAWSGardenerInput(enableIMDSv2 bool) *gqlschema.AWSProviderConfigInput {
	return &gqlschema.AWSProviderConfigInput{
		AwsZones: []*gqlschema.AWSZoneInput{
			{
				Name:         "zone",
				PublicCidr:   "10.10.11.12/255",
				InternalCidr: "10.10.11.13/255",
				WorkerCidr:   "10.10.11.12/255",
			},
		},
		VpcCidr:      "10.10.11.11/255",
		EnableIMDSv2: &enableIMDSv2,
	}
}

func fixGCPGardenerInput(zones []string) *gqlschema.GCPProviderConfigInput {
	return &gqlschema.GCPProviderConfigInput{Zones: zones}
}

func fixAzureGardenerInput(zones []string, enableNAT *bool) *gqlschema.AzureProviderConfigInput {
	return &gqlschema.AzureProviderConfigInput{VnetCidr: "10.10.11.11/255", Zones: zones, EnableNatGateway: enableNAT, IdleConnectionTimeoutMinutes: util.PtrTo(4)}
}

func fixAzureZoneSubnetsInput(enableNAT bool) *gqlschema.AzureProviderConfigInput {
	return &gqlschema.AzureProviderConfigInput{
		VnetCidr:                     "10.10.11.11/255",
		EnableNatGateway:             util.PtrTo(enableNAT),
		IdleConnectionTimeoutMinutes: util.PtrTo(4),
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

func fixWorker(zones []string, providerConfig *apimachineryRuntime.RawExtension) gardener_types.Worker {
	return gardener_types.Worker{
		Name:           "cpu-worker-0",
		MaxSurge:       util.PtrTo(intstr.FromInt(30)),
		MaxUnavailable: util.PtrTo(intstr.FromInt(1)),
		Machine: gardener_types.Machine{
			Type: "machine",
			Image: &gardener_types.ShootMachineImage{
				Name:    "gardenlinux",
				Version: util.PtrTo("25.0.0"),
			},
		},
		Volume: &gardener_types.Volume{
			Type:       util.PtrTo("SSD"),
			VolumeSize: "30Gi",
		},
		Maximum:        3,
		Minimum:        1,
		Zones:          zones,
		ProviderConfig: providerConfig,
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
