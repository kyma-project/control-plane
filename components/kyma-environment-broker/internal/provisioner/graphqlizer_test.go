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
        autoScalerMin: 0,
        autoScalerMax: 0,
        maxSurge: 0,
		maxUnavailable: 0,
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
			name: "Azure will all parameters",
			givenInput: gqlschema.AzureProviderConfigInput{
				VnetCidr: "8.8.8.8",
				Zones:    []string{"fix-az-zone-1", "fix-az-zone-2"},
			},
			expected: `{
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

func Test_UpgradeShootInputToGraphQL(t *testing.T) {
	// given
	sut := Graphqlizer{}
	exp := `{
    gardenerConfig: {
      kubernetesVersion: "1.18.0",
      machineImage: "gardenlinux",
      machineImageVersion: "184.0.0",
      enableKubernetesVersionAutoUpdate: true,
      enableMachineImageVersionAutoUpdate: false,
    }
  }`

	// when
	got, err := sut.UpgradeShootInputToGraphQL(gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion:                   strPrt("1.18.0"),
			MachineImage:                        strPrt("gardenlinux"),
			MachineImageVersion:                 strPrt("184.0.0"),
			EnableKubernetesVersionAutoUpdate:   boolPtr(true),
			EnableMachineImageVersionAutoUpdate: boolPtr(false),
		},
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

func strPrt(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
