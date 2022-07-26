package provisioner

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKymaConfigToGraphQLAllParametersProvided(t *testing.T) {
	// given
	profile := gqlschema.KymaProfileProduction
	strategy := gqlschema.ConflictStrategyReplace
	fixInput := gqlschema.KymaConfigInput{
		Version:          "966",
		Profile:          &profile,
		ConflictStrategy: &strategy,
		Components: []*gqlschema.ComponentConfigurationInput{
			{
				Component: "pico",
				Namespace: "bello",
			},
			{
				Component:        "custom-component",
				Namespace:        "bello",
				ConflictStrategy: &strategy,
				SourceURL:        ptr.String("github.com/kyma-incubator/custom-component"),
			},
			{
				Component: "hakuna",
				Namespace: "matata",
				Configuration: []*gqlschema.ConfigEntryInput{
					{
						Key:    "testing-secret-key",
						Value:  "testing-secret-value",
						Secret: ptr.Bool(true),
					},
					{
						Key:   "testing-public-key",
						Value: "testing-public-value\nmultiline",
					},
				},
			},
		},
		Configuration: []*gqlschema.ConfigEntryInput{
			{
				Key:   "important-global-override",
				Value: "false",
			},
			{
				Key:    "ultimate.answer",
				Value:  "42",
				Secret: ptr.Bool(true),
			},
		},
	}
	expRender := `{
		version: "966",
		profile: Production,
		conflictStrategy: Replace,
		components: [
			{
				component: "pico",
				namespace: "bello",
			}
			{
				component: "custom-component",
				namespace: "bello",
				sourceURL: "github.com/kyma-incubator/custom-component",
				conflictStrategy: Replace,
			}
			{
				component: "hakuna",
				namespace: "matata",
				configuration: [
					{
						key: "testing-secret-key",
						value: "testing-secret-value",
						secret: true,
					}
					{
						key: "testing-public-key",
						value: "testing-public-value\nmultiline",
					}
				]
			}
		]
		configuration: [
			{
				key: "important-global-override",
				value: "false",
			}
			{
				key: "ultimate.answer",
				value: "42",
				secret: true,
			}
		]
	}`

	sut := Graphqlizer{}

	// when
	gotRender, err := sut.KymaConfigToGraphQL(fixInput)

	// then
	require.NoError(t, err)

	assert.Equal(t, expRender, gotRender)
}

func TestKymaConfigToGraphQLOnlyKymaVersionAndProfile(t *testing.T) {
	// given
	profile := gqlschema.KymaProfileEvaluation
	fixInput := gqlschema.KymaConfigInput{
		Version: "966",
		Profile: &profile,
	}
	expRender := `{
		version: "966",
		profile: Evaluation,
	}`

	sut := Graphqlizer{}

	// when
	gotRender, err := sut.KymaConfigToGraphQL(fixInput)

	// then
	require.NoError(t, err)

	assert.Equal(t, expRender, gotRender)
}

func Test_GardenerConfigInputToGraphQL(t *testing.T) {
	// given
	sut := Graphqlizer{}
	exp := `{
		name: "c-90a3016",
		kubernetesVersion: "1.18",
		volumeSizeGB: 50,
		machineType: "Standard_D4_v3",
		region: "europe",
		provider: "Azure",
		diskType: "Standard_LRS",
		targetSecret: "scr",
		workerCidr: "10.250.0.0/19",
		autoScalerMin: 2,
		autoScalerMax: 4,
		maxSurge: 4,
		maxUnavailable: 1,
		enableKubernetesVersionAutoUpdate: true,
		enableMachineImageVersionAutoUpdate: true,
	}`

	// when
	name := "c-90a3016"
	got, err := sut.GardenerConfigInputToGraphQL(gqlschema.GardenerConfigInput{
		Name:                                name,
		Region:                              "europe",
		VolumeSizeGb:                        ptr.Integer(50),
		WorkerCidr:                          "10.250.0.0/19",
		Provider:                            "Azure",
		DiskType:                            ptr.String("Standard_LRS"),
		TargetSecret:                        "scr",
		MachineType:                         "Standard_D4_v3",
		KubernetesVersion:                   "1.18",
		AutoScalerMin:                       2,
		AutoScalerMax:                       4,
		MaxSurge:                            4,
		MaxUnavailable:                      1,
		EnableKubernetesVersionAutoUpdate:   ptr.Bool(true),
		EnableMachineImageVersionAutoUpdate: ptr.Bool(true),
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, exp, got)
}

func Test_GardenerConfigInputToGraphQLWithOIDC(t *testing.T) {
	// given
	sut := Graphqlizer{}
	exp := `{
		name: "c-90a3016",
		kubernetesVersion: "1.18",
		volumeSizeGB: 50,
		machineType: "Standard_D4_v3",
		region: "europe",
		provider: "Azure",
		diskType: "Standard_LRS",
		targetSecret: "scr",
		workerCidr: "10.250.0.0/19",
		autoScalerMin: 0,
		autoScalerMax: 0,
		maxSurge: 0,
		maxUnavailable: 0,
		oidcConfig: {
			clientID: "client-id",
			issuerURL: "https://issuer.url",
			groupsClaim: "",
			signingAlgs: [],
			usernameClaim: "",
			usernamePrefix: "",
		}
	}`

	// when
	name := "c-90a3016"
	got, err := sut.GardenerConfigInputToGraphQL(gqlschema.GardenerConfigInput{
		Name:              name,
		Region:            "europe",
		VolumeSizeGb:      ptr.Integer(50),
		WorkerCidr:        "10.250.0.0/19",
		Provider:          "Azure",
		DiskType:          ptr.String("Standard_LRS"),
		TargetSecret:      "scr",
		MachineType:       "Standard_D4_v3",
		KubernetesVersion: "1.18",
		OidcConfig: &gqlschema.OIDCConfigInput{
			ClientID:       "client-id",
			GroupsClaim:    "",
			IssuerURL:      "https://issuer.url",
			SigningAlgs:    nil,
			UsernameClaim:  "",
			UsernamePrefix: "",
		},
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, exp, got)
}

func Test_GardenerConfigInputToGraphQLWithDNS(t *testing.T) {
	// given
	sut := Graphqlizer{}
	exp := `{
		name: "c-90a3016",
		kubernetesVersion: "1.18",
		volumeSizeGB: 50,
		machineType: "Standard_D4_v3",
		region: "europe",
		provider: "Azure",
		diskType: "Standard_LRS",
		targetSecret: "scr",
		workerCidr: "10.250.0.0/19",
		autoScalerMin: 0,
		autoScalerMax: 0,
		maxSurge: 0,
		maxUnavailable: 0,
		oidcConfig: {
			clientID: "client-id",
			issuerURL: "https://issuer.url",
			groupsClaim: "",
			signingAlgs: [],
			usernameClaim: "",
			usernamePrefix: "",
		}
		dnsConfig: {
			domain: "c-90a3016.kymatest.ondemand.com",
			providers: [
				{
					domainsInclude: ["kymatest.ondemand.com"],
					primary: true,
					secretName: "aws-secret-for-custom-domain",
					type: "aws-route53",
				}
			]
		}
	}`

	// when
	name := "c-90a3016"
	got, err := sut.GardenerConfigInputToGraphQL(gqlschema.GardenerConfigInput{
		Name:              name,
		Region:            "europe",
		VolumeSizeGb:      ptr.Integer(50),
		WorkerCidr:        "10.250.0.0/19",
		Provider:          "Azure",
		DiskType:          ptr.String("Standard_LRS"),
		TargetSecret:      "scr",
		MachineType:       "Standard_D4_v3",
		KubernetesVersion: "1.18",
		OidcConfig: &gqlschema.OIDCConfigInput{
			ClientID:       "client-id",
			GroupsClaim:    "",
			IssuerURL:      "https://issuer.url",
			SigningAlgs:    nil,
			UsernameClaim:  "",
			UsernamePrefix: "",
		},
		DNSConfig: &gqlschema.DNSConfigInput{
			Domain: name + ".kymatest.ondemand.com",
			Providers: []*gqlschema.DNSProviderInput{
				&gqlschema.DNSProviderInput{
					DomainsInclude: []string{"kymatest.ondemand.com"},
					Primary:        true,
					SecretName:     "aws-secret-for-custom-domain",
					Type:           "aws-route53",
				},
			},
		},
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, exp, got)
}

func Test_GardenerConfigInputToGraphQLWithMachineImage(t *testing.T) {
	// given
	sut := Graphqlizer{}
	exp := `{
		name: "c-90a3016",
		kubernetesVersion: "1.18",
		volumeSizeGB: 50,
		machineType: "Standard_D4_v3",
		machineImage: "coreos",
		machineImageVersion: "255.0",
		region: "europe",
		provider: "Azure",
		diskType: "Standard_LRS",
		targetSecret: "scr",
		workerCidr: "10.250.0.0/19",
		autoScalerMin: 0,
		autoScalerMax: 0,
		maxSurge: 0,
		maxUnavailable: 0,
	}`

	// when
	name := "c-90a3016"
	got, err := sut.GardenerConfigInputToGraphQL(gqlschema.GardenerConfigInput{
		Name:                name,
		Region:              "europe",
		VolumeSizeGb:        ptr.Integer(50),
		WorkerCidr:          "10.250.0.0/19",
		Provider:            "Azure",
		DiskType:            ptr.String("Standard_LRS"),
		TargetSecret:        "scr",
		MachineType:         "Standard_D4_v3",
		KubernetesVersion:   "1.18",
		MachineImage:        strPrt("coreos"),
		MachineImageVersion: strPrt("255.0"),
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, exp, got)
}

func Test_LabelsToGQL(t *testing.T) {

	sut := Graphqlizer{}

	for _, testCase := range []struct {
		description string
		input       gqlschema.Labels
		expected    string
	}{
		{
			description: "string labels",
			input: gqlschema.Labels{
				"test": "966",
			},
			expected: `{test:"966",}`,
		},
		{
			description: "string array labels",
			input: gqlschema.Labels{
				"test": []string{"966"},
			},
			expected: `{test:["966"],}`,
		},
		{
			description: "string array labels",
			input: gqlschema.Labels{
				"test": map[string]string{"abcd": "966"},
			},
			expected: `{test:{abcd:"966",},}`,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// when
			render, err := sut.LabelsToGQL(testCase.input)

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, render)
		})
	}
}

func TestAzureProviderConfigInputToGraphQL(t *testing.T) {
	tests := []struct {
		name       string
		givenInput gqlschema.AzureProviderConfigInput
		expected   string
	}{
		{
			name: "Azure with zones parameter",
			givenInput: gqlschema.AzureProviderConfigInput{
				EnableNatGateway: boolPtr(true),
				VnetCidr:         "8.8.8.8",
				Zones:            []string{"fix-az-zone-1", "fix-az-zone-2"},
			},
			expected: `{
		enableNatGateway: true,
		vnetCidr: "8.8.8.8",
		zones: ["fix-az-zone-1","fix-az-zone-2"],
	}`,
		},
		{
			name: "Azure with no zones passed",
			givenInput: gqlschema.AzureProviderConfigInput{
				VnetCidr: "8.8.8.8",
			},
			expected: `{
		vnetCidr: "8.8.8.8",
	}`,
		},
		{
			name: "Azure with azureZones passed",
			givenInput: gqlschema.AzureProviderConfigInput{
				VnetCidr: "8.8.8.8",
				AzureZones: []*gqlschema.AzureZoneInput{
					{
						Name: 1,
						Cidr: "10.250.0.0/19",
					},
					{
						Name: 2,
						Cidr: "10.250.32.0/19",
					},
				},
			},
			expected: `{
		vnetCidr: "8.8.8.8",
		azureZones: [
			{
				name: 1,
				cidr: "10.250.0.0/19",
			}
			{
				name: 2,
				cidr: "10.250.32.0/19",
			}
		]
	}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Graphqlizer{}

			// when
			got, err := g.AzureProviderConfigInputToGraphQL(tt.givenInput)

			// then
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGCPProviderConfigInputToGraphQL(t *testing.T) {
	// given
	fixInput := gqlschema.GCPProviderConfigInput{
		Zones: []string{"fix-gcp-zone-1", "fix-gcp-zone-2"},
	}
	expected := `{ zones: ["fix-gcp-zone-1","fix-gcp-zone-2"] }`
	g := &Graphqlizer{}

	// when
	got, err := g.GCPProviderConfigInputToGraphQL(fixInput)

	// then
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestAWSProviderConfigInputToGraphQL(t *testing.T) {
	tests := []struct {
		name       string
		givenInput gqlschema.AWSProviderConfigInput
		expected   string
	}{
		{
			name: "AWS will all parameters",
			givenInput: gqlschema.AWSProviderConfigInput{
				VpcCidr: "10.250.0.0/16",
				AwsZones: []*gqlschema.AWSZoneInput{
					{
						Name:         "eu-central-1a",
						WorkerCidr:   "10.250.0.0/22",
						PublicCidr:   "10.250.20.0/22",
						InternalCidr: "10.250.40.0/22",
					},
					{
						Name:         "eu-central-1b",
						WorkerCidr:   "10.250.4.0/22",
						PublicCidr:   "10.250.24.0/22",
						InternalCidr: "10.250.44.0/22",
					},
				},
			},
			expected: `{
		vpcCidr: "10.250.0.0/16",
		awsZones: [
			{
				name: "eu-central-1a",
				workerCidr: "10.250.0.0/22",
				publicCidr: "10.250.20.0/22",
				internalCidr: "10.250.40.0/22",
			}
			{
				name: "eu-central-1b",
				workerCidr: "10.250.4.0/22",
				publicCidr: "10.250.24.0/22",
				internalCidr: "10.250.44.0/22",
			}
		]
	}`,
		},
		{
			name: "AWS with no zones passed",
			givenInput: gqlschema.AWSProviderConfigInput{
				VpcCidr: "8.8.8.8",
			},
			expected: `{
		vpcCidr: "8.8.8.8",
	}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Graphqlizer{}

			// when
			got, err := g.AWSProviderConfigInputToGraphQL(tt.givenInput)

			// then
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func Test_UpgradeShootInputToGraphQL(t *testing.T) {
	// given
	sut := Graphqlizer{}
	exp := `{
	gardenerConfig: {
		kubernetesVersion: "1.18.0",
		machineType: "m5.xlarge",
		machineImage: "gardenlinux",
		machineImageVersion: "184.0.0",
		autoScalerMin: 2,
		autoScalerMax: 4,
		maxSurge: 4,
		maxUnavailable: 1,
		enableKubernetesVersionAutoUpdate: true,
		enableMachineImageVersionAutoUpdate: false,
		oidcConfig: {
			clientID: "cid",
			issuerURL: "issuer.url",
			groupsClaim: "groups",
			signingAlgs: ["RSA256"],
			usernameClaim: "sub",
			usernamePrefix: "-",
		},
	},
	administrators: ["newAdmin@kyma.cx"],
}`

	// when
	got, err := sut.UpgradeShootInputToGraphQL(gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion:                   strPrt("1.18.0"),
			MachineType:                         strPrt("m5.xlarge"),
			MachineImage:                        strPrt("gardenlinux"),
			MachineImageVersion:                 strPrt("184.0.0"),
			EnableKubernetesVersionAutoUpdate:   boolPtr(true),
			EnableMachineImageVersionAutoUpdate: boolPtr(false),
			AutoScalerMin:                       ptr.Integer(2),
			AutoScalerMax:                       ptr.Integer(4),
			MaxSurge:                            ptr.Integer(4),
			MaxUnavailable:                      ptr.Integer(1),
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "cid",
				GroupsClaim:    "groups",
				IssuerURL:      "issuer.url",
				SigningAlgs:    []string{"RSA256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
		},
		Administrators: []string{"newAdmin@kyma.cx"},
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, exp, got)
}

func TestOpenstack(t *testing.T) {
	// given
	input := gqlschema.ProviderSpecificInput{
		OpenStackConfig: &gqlschema.OpenStackProviderConfigInput{
			Zones:                []string{"z1"},
			FloatingPoolName:     "fp",
			CloudProfileName:     "cp",
			LoadBalancerProvider: "lbp",
		},
	}

	g := &Graphqlizer{}

	// when
	got, err := g.GardenerConfigInputToGraphQL(gqlschema.GardenerConfigInput{
		ExposureClassName:      ptr.String("converged-cloud-internet"),
		ProviderSpecificConfig: &input,
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, `{
		kubernetesVersion: "",
		machineType: "",
		region: "",
		provider: "",
		targetSecret: "",
		workerCidr: "",
		autoScalerMin: 0,
		autoScalerMax: 0,
		maxSurge: 0,
		maxUnavailable: 0,
		exposureClassName: "converged-cloud-internet",
		providerSpecificConfig: {
			openStackConfig: {
				zones: ["z1"],
				floatingPoolName: "fp",
				cloudProfileName: "cp",
				loadBalancerProvider: "lbp"
			},
		}
	}`, got)

}

func Test_ClusterConfigToGraphQL(t *testing.T) {
	tests := []struct {
		name       string
		givenInput gqlschema.ClusterConfigInput
		expected   string
	}{
		{
			name: "Cluster config with administrators",
			givenInput: gqlschema.ClusterConfigInput{
				Administrators: []string{"test@test.pl"},
			},
			expected: `{
		administrators: ["test@test.pl"],
	}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Graphqlizer{}

			// when
			got, err := g.ClusterConfigToGraphQL(tt.givenInput)

			// then
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func strPrt(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
