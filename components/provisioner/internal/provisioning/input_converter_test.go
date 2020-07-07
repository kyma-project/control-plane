package provisioning

import (
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/internal/uuid"

	"github.com/kyma-project/control-plane/components/provisioner/internal/installation/release"
	realeaseMocks "github.com/kyma-project/control-plane/components/provisioner/internal/installation/release/mocks"

	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"

	"github.com/kyma-project/control-plane/components/provisioner/internal/uuid/mocks"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/assert"
)

const (
	kymaVersion                   = "1.5"
	clusterEssentialsComponent    = "cluster-essentials"
	coreComponent                 = "core"
	rafterComponent               = "rafter"
	applicationConnectorComponent = "application-connector"

	rafterSourceURL = "github.com/kyma-project/kyma.git//resources/rafter"

	gardenerProject = "gardener-project"
)

func Test_ProvisioningInputToCluster(t *testing.T) {

	readSession := &realeaseMocks.Repository{}
	readSession.On("GetReleaseByVersion", kymaVersion).Return(fixKymaRelease(), nil)

	gcpGardenerProvider := &gqlschema.GCPProviderConfigInput{Zones: []string{"fix-gcp-zone-1", "fix-gcp-zone-2"}}

	gardenerGCPGQLInput := gqlschema.ProvisionRuntimeInput{
		RuntimeInput: &gqlschema.RuntimeInput{
			Name:        "runtimeName",
			Description: nil,
			Labels:      &gqlschema.Labels{},
		},
		ClusterConfig: &gqlschema.ClusterConfigInput{
			GardenerConfig: &gqlschema.GardenerConfigInput{
				KubernetesVersion: "version",
				VolumeSizeGb:      1024,
				MachineType:       "n1-standard-1",
				Region:            "region",
				Provider:          "GCP",
				Purpose:           util.StringPtr("testing"),
				Seed:              util.StringPtr("gcp-eu1"),
				TargetSecret:      "secret",
				DiskType:          "ssd",
				WorkerCidr:        "cidr",
				AutoScalerMin:     1,
				AutoScalerMax:     5,
				MaxSurge:          1,
				MaxUnavailable:    2,
				ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
					GcpConfig: gcpGardenerProvider,
				},
			},
		},
		KymaConfig: fixKymaGraphQLConfigInput(),
	}

	expectedGCPProviderCfg, err := model.NewGCPGardenerConfig(gcpGardenerProvider)
	require.NoError(t, err)

	expectedGardenerGCPRuntimeConfig := model.Cluster{
		ID: "runtimeID",
		ClusterConfig: model.GardenerConfig{
			ID:                     "id",
			Name:                   "verylon",
			ProjectName:            gardenerProject,
			MachineType:            "n1-standard-1",
			Region:                 "region",
			KubernetesVersion:      "version",
			VolumeSizeGB:           1024,
			DiskType:               "ssd",
			Provider:               "GCP",
			Purpose:                util.StringPtr("testing"),
			Seed:                   "gcp-eu1",
			TargetSecret:           "secret",
			WorkerCidr:             "cidr",
			AutoScalerMin:          1,
			AutoScalerMax:          5,
			MaxSurge:               1,
			MaxUnavailable:         2,
			ClusterID:              "runtimeID",
			GardenerProviderConfig: expectedGCPProviderCfg,
		},
		Kubeconfig:   nil,
		KymaConfig:   fixKymaConfig(),
		Tenant:       tenant,
		SubAccountId: util.StringPtr(subAccountId),
	}

	createGQLRuntimeInputAzure := func(zones []string) gqlschema.ProvisionRuntimeInput {
		return gqlschema.ProvisionRuntimeInput{
			RuntimeInput: &gqlschema.RuntimeInput{
				Name:        "runtimeName",
				Description: nil,
				Labels:      &gqlschema.Labels{},
			},
			ClusterConfig: &gqlschema.ClusterConfigInput{
				GardenerConfig: &gqlschema.GardenerConfigInput{
					KubernetesVersion: "version",
					VolumeSizeGb:      1024,
					MachineType:       "n1-standard-1",
					Region:            "region",
					Provider:          "Azure",
					Purpose:           util.StringPtr("testing"),
					TargetSecret:      "secret",
					DiskType:          "ssd",
					WorkerCidr:        "cidr",
					AutoScalerMin:     1,
					AutoScalerMax:     5,
					MaxSurge:          1,
					MaxUnavailable:    2,
					ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
						AzureConfig: &gqlschema.AzureProviderConfigInput{
							VnetCidr: "cidr",
							Zones:    zones,
						},
					},
				},
			},
			KymaConfig: fixKymaGraphQLConfigInput(),
		}
	}

	expectedGardenerAzureRuntimeConfig := func(zones []string) model.Cluster {

		expectedAzureProviderCfg, err := model.NewAzureGardenerConfig(&gqlschema.AzureProviderConfigInput{VnetCidr: "cidr", Zones: zones})
		require.NoError(t, err)

		return model.Cluster{
			ID: "runtimeID",
			ClusterConfig: model.GardenerConfig{
				ID:                     "id",
				Name:                   "verylon",
				ProjectName:            gardenerProject,
				MachineType:            "n1-standard-1",
				Region:                 "region",
				KubernetesVersion:      "version",
				VolumeSizeGB:           1024,
				DiskType:               "ssd",
				Provider:               "Azure",
				Purpose:                util.StringPtr("testing"),
				Seed:                   "",
				TargetSecret:           "secret",
				WorkerCidr:             "cidr",
				AutoScalerMin:          1,
				AutoScalerMax:          5,
				MaxSurge:               1,
				MaxUnavailable:         2,
				ClusterID:              "runtimeID",
				GardenerProviderConfig: expectedAzureProviderCfg,
			},
			Kubeconfig:   nil,
			KymaConfig:   fixKymaConfig(),
			Tenant:       tenant,
			SubAccountId: util.StringPtr(subAccountId),
		}
	}

	awsGardenerProvider := &gqlschema.AWSProviderConfigInput{
		Zone:         "zone",
		InternalCidr: "cidr",
		VpcCidr:      "cidr",
		PublicCidr:   "cidr",
	}

	gardenerAWSGQLInput := gqlschema.ProvisionRuntimeInput{
		RuntimeInput: &gqlschema.RuntimeInput{
			Name:        "runtimeName",
			Description: nil,
			Labels:      &gqlschema.Labels{},
		},
		ClusterConfig: &gqlschema.ClusterConfigInput{
			GardenerConfig: &gqlschema.GardenerConfigInput{
				KubernetesVersion: "version",
				VolumeSizeGb:      1024,
				MachineType:       "n1-standard-1",
				Region:            "region",
				Provider:          "AWS",
				Purpose:           util.StringPtr("testing"),
				Seed:              util.StringPtr("aws-eu1"),
				TargetSecret:      "secret",
				DiskType:          "ssd",
				WorkerCidr:        "cidr",
				AutoScalerMin:     1,
				AutoScalerMax:     5,
				MaxSurge:          1,
				MaxUnavailable:    2,
				ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
					AwsConfig: awsGardenerProvider,
				},
			},
		},
		KymaConfig: fixKymaGraphQLConfigInput(),
	}

	expectedAWSProviderCfg, err := model.NewAWSGardenerConfig(awsGardenerProvider)
	require.NoError(t, err)

	expectedGardenerAWSRuntimeConfig := model.Cluster{
		ID: "runtimeID",
		ClusterConfig: model.GardenerConfig{
			ID:                     "id",
			Name:                   "verylon",
			ProjectName:            gardenerProject,
			MachineType:            "n1-standard-1",
			Region:                 "region",
			KubernetesVersion:      "version",
			VolumeSizeGB:           1024,
			DiskType:               "ssd",
			Provider:               "AWS",
			Purpose:                util.StringPtr("testing"),
			Seed:                   "aws-eu1",
			TargetSecret:           "secret",
			WorkerCidr:             "cidr",
			AutoScalerMin:          1,
			AutoScalerMax:          5,
			MaxSurge:               1,
			MaxUnavailable:         2,
			ClusterID:              "runtimeID",
			GardenerProviderConfig: expectedAWSProviderCfg,
		},
		Kubeconfig:   nil,
		KymaConfig:   fixKymaConfig(),
		Tenant:       tenant,
		SubAccountId: util.StringPtr(subAccountId),
	}

	gardenerZones := []string{"fix-az-zone-1", "fix-az-zone-2"}

	configurations := []struct {
		input       gqlschema.ProvisionRuntimeInput
		expected    model.Cluster
		description string
	}{
		{
			input:       gardenerGCPGQLInput,
			expected:    expectedGardenerGCPRuntimeConfig,
			description: "Should create proper runtime config struct with Gardener input for GCP provider",
		},
		{
			input:       createGQLRuntimeInputAzure(nil),
			expected:    expectedGardenerAzureRuntimeConfig(nil),
			description: "Should create proper runtime config struct with Gardener input for Azure provider",
		},
		{
			input:       createGQLRuntimeInputAzure(gardenerZones),
			expected:    expectedGardenerAzureRuntimeConfig(gardenerZones),
			description: "Should create proper runtime config struct with Gardener input for Azure provider with zones passed",
		},
		{
			input:       gardenerAWSGQLInput,
			expected:    expectedGardenerAWSRuntimeConfig,
			description: "Should create proper runtime config struct with Gardener input for AWS provider",
		},
	}

	for _, testCase := range configurations {
		t.Run(testCase.description, func(t *testing.T) {
			//given
			uuidGeneratorMock := &mocks.UUIDGenerator{}
			uuidGeneratorMock.On("New").Return("id").Times(6)
			uuidGeneratorMock.On("New").Return("very-Long-ID-That-Has-More-Than-Fourteen-Characters-And-Even-Some-Hyphens")

			inputConverter := NewInputConverter(uuidGeneratorMock, readSession, gardenerProject)

			//when
			runtimeConfig, err := inputConverter.ProvisioningInputToCluster("runtimeID", testCase.input, tenant, subAccountId)

			//then
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, runtimeConfig)
			uuidGeneratorMock.AssertExpectations(t)
		})
	}
}

func TestConverter_ProvisioningInputToCluster_Error(t *testing.T) {

	t.Run("should return error when failed to get kyma release", func(t *testing.T) {
		// given
		uuidGeneratorMock := &mocks.UUIDGenerator{}
		readSession := &realeaseMocks.Repository{}
		readSession.On("GetReleaseByVersion", kymaVersion).Return(model.Release{}, dberrors.NotFound("error"))

		input := gqlschema.ProvisionRuntimeInput{
			ClusterConfig: &gqlschema.ClusterConfigInput{
				GardenerConfig: &gqlschema.GardenerConfigInput{},
			},
			KymaConfig: &gqlschema.KymaConfigInput{
				Version: kymaVersion,
			},
		}

		inputConverter := NewInputConverter(uuidGeneratorMock, readSession, gardenerProject)

		//when
		_, err := inputConverter.ProvisioningInputToCluster("runtimeID", input, tenant, subAccountId)

		//then
		require.Error(t, err)
		uuidGeneratorMock.AssertExpectations(t)
	})

	t.Run("should return error when Cluster Config not provided", func(t *testing.T) {
		// given
		input := gqlschema.ProvisionRuntimeInput{}

		inputConverter := NewInputConverter(nil, nil, gardenerProject)

		//when
		_, err := inputConverter.ProvisioningInputToCluster("runtimeID", input, tenant, subAccountId)

		//then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ClusterConfig not provided")
	})

	t.Run("should return error when Gardener Config not provided", func(t *testing.T) {
		// given
		input := gqlschema.ProvisionRuntimeInput{
			ClusterConfig: &gqlschema.ClusterConfigInput{
				GardenerConfig: nil,
			},
		}

		inputConverter := NewInputConverter(nil, nil, gardenerProject)

		//when
		_, err := inputConverter.ProvisioningInputToCluster("runtimeID", input, tenant, subAccountId)

		//then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GardenerConfig not provided")
	})

	t.Run("should return error when no Gardener provider specified", func(t *testing.T) {
		// given
		uuidGeneratorMock := &mocks.UUIDGenerator{}
		uuidGeneratorMock.On("New").Return("id").Times(4)

		input := gqlschema.ProvisionRuntimeInput{
			ClusterConfig: &gqlschema.ClusterConfigInput{
				GardenerConfig: &gqlschema.GardenerConfigInput{},
			},
		}

		inputConverter := NewInputConverter(uuidGeneratorMock, nil, gardenerProject)

		//when
		_, err := inputConverter.ProvisioningInputToCluster("runtimeID", input, tenant, subAccountId)

		//then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider config not specified")
	})

}

func Test_UpgradeShootInputToCluster(t *testing.T) {
	evaluationPurpose := "evaluation"
	testingPurpose := "testing"

	readSession := &realeaseMocks.Repository{}
	readSession.On("GetReleaseByVersion", kymaVersion).Return(fixKymaRelease(), nil)

	expectedGCPProviderConfig, err := model.NewGCPGardenerConfig(
		&gqlschema.GCPProviderConfigInput{
			Zones: []string{"fix-gcp-zone-1", "fix-gcp-zone-2"},
		},
	)
	require.NoError(t, err)

	configurations := []struct {
		description    string
		upgradeInput   gqlschema.UpgradeShootInput
		currentConfig  model.GardenerConfig
		upgradedConfig model.GardenerConfig
	}{
		{description: "GCP shoot upgrade",
			upgradeInput: newUpgradeShootInput(),
			currentConfig: model.GardenerConfig{
				KubernetesVersion: "version",
				VolumeSizeGB:      1,
				DiskType:          "ssd",
				MachineType:       "1",
				Purpose:           &evaluationPurpose,
				WorkerCidr:        "cidr",
				AutoScalerMin:     1,
				AutoScalerMax:     2,
				MaxSurge:          1,
				MaxUnavailable:    1,
			},
			upgradedConfig: model.GardenerConfig{
				KubernetesVersion: "version2",
				VolumeSizeGB:      50,
				DiskType:          "papyrus",
				MachineType:       "new-machine",
				Purpose:           &testingPurpose,
				WorkerCidr:        "cidr2",
				AutoScalerMin:     2,
				AutoScalerMax:     6,
				MaxSurge:          2,
				MaxUnavailable:    1,
			},
		},
		{description: "Azure shoot upgrade",
			upgradeInput: newAzureUpgradeShootInput(),
			currentConfig: model.GardenerConfig{
				KubernetesVersion: "version",
				VolumeSizeGB:      1,
				DiskType:          "ssd",
				MachineType:       "1",
				Purpose:           &evaluationPurpose,
				WorkerCidr:        "cidr",
				AutoScalerMin:     1,
				AutoScalerMax:     2,
				MaxSurge:          1,
				MaxUnavailable:    1,
				GardenerProviderConfig: &model.AzureGardenerConfig{
					ProviderSpecificConfig: model.ProviderSpecificConfig("config"),
				},
			},
			upgradedConfig: model.GardenerConfig{
				KubernetesVersion: "version2",
				VolumeSizeGB:      50,
				DiskType:          "papyrus",
				MachineType:       "new-machine",
				Purpose:           &testingPurpose,
				WorkerCidr:        "cidr2",
				AutoScalerMin:     2,
				AutoScalerMax:     6,
				MaxSurge:          2,
				MaxUnavailable:    1,
				GardenerProviderConfig: &model.AzureGardenerConfig{
					ProviderSpecificConfig: model.ProviderSpecificConfig("config"),
				},
			},
		},
		{description: "AWS shoot upgrade",
			upgradeInput:   gqlschema.UpgradeShootInput{},
			currentConfig:  model.GardenerConfig{},
			upgradedConfig: model.GardenerConfig{},
		},
	}

	for _, testCase := range configurations {
		t.Run(testCase.description, func(t *testing.T) {
			//given
			uuidGeneratorMock := &mocks.UUIDGenerator{}
			uuidGeneratorMock.On("New").Return("id").Times(6)
			uuidGeneratorMock.On("New").Return("very-Long-ID-That-Has-More-Than-Fourteen-Characters-And-Even-Some-Hyphens")

			inputConverter := NewInputConverter(uuidGeneratorMock, readSession, gardenerProject)

			//when
			runtimeConfig, err := inputConverter.UpgradeShootInputToGardenerConfig(testCase.upgradeInput, testCase.currentConfig)

			//then
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, runtimeConfig)
			uuidGeneratorMock.AssertExpectations(t)
		})
	}
}

func newInputConverterTester(uuidGenerator uuid.UUIDGenerator, releaseRepo release.Provider) *converter {
	return &converter{
		uuidGenerator: uuidGenerator,
		releaseRepo:   releaseRepo,
	}
}

func newUpgradeShootInput() gqlschema.UpgradeShootInput {
	newKubernetesVersion := "version2"
	newMachineType := "new-machine"
	newDiskType := "papyrus"
	newVolumeSizeGb := 50
	newCidr := "cidr2"

	return gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion:      &newKubernetesVersion,
			MachineType:            &newMachineType,
			DiskType:               &newDiskType,
			VolumeSizeGb:           &newVolumeSizeGb,
			WorkerCidr:             &newCidr,
			AutoScalerMin:          util.IntPtr(2),
			AutoScalerMax:          util.IntPtr(6),
			MaxSurge:               util.IntPtr(2),
			MaxUnavailable:         util.IntPtr(1),
			ProviderSpecificConfig: nil,
		},
	}
}

func newAzureUpgradeShootInput() gqlschema.UpgradeShootInput {
	input := newUpgradeShootInput()
	input.GardenerConfig.ProviderSpecificConfig = &gqlschema.ProviderSpecificInput{
		AzureConfig: &gqlschema.AzureProviderConfigInput{
			VnetCidr: "cidr2",
		},
	}
	return input
}

func fixKymaGraphQLConfigInput() *gqlschema.KymaConfigInput {
	return &gqlschema.KymaConfigInput{
		Version: kymaVersion,
		Components: []*gqlschema.ComponentConfigurationInput{
			{
				Component: clusterEssentialsComponent,
				Namespace: kymaSystemNamespace,
			},
			{
				Component: coreComponent,
				Namespace: kymaSystemNamespace,
				Configuration: []*gqlschema.ConfigEntryInput{
					fixGQLConfigEntryInput("test.config.key", "value", util.BoolPtr(false)),
					fixGQLConfigEntryInput("test.config.key2", "value2", util.BoolPtr(false)),
				},
			},
			{
				Component: rafterComponent,
				Namespace: kymaSystemNamespace,
				SourceURL: util.StringPtr(rafterSourceURL),
			},
			{
				Component: applicationConnectorComponent,
				Namespace: kymaIntegrationNamespace,
				Configuration: []*gqlschema.ConfigEntryInput{
					fixGQLConfigEntryInput("test.config.key", "value", util.BoolPtr(false)),
					fixGQLConfigEntryInput("test.secret.key", "secretValue", util.BoolPtr(true)),
				},
			},
		},
		Configuration: []*gqlschema.ConfigEntryInput{
			fixGQLConfigEntryInput("global.config.key", "globalValue", util.BoolPtr(false)),
			fixGQLConfigEntryInput("global.config.key2", "globalValue2", util.BoolPtr(false)),
			fixGQLConfigEntryInput("global.secret.key", "globalSecretValue", util.BoolPtr(true)),
		},
	}
}

func fixGQLConfigEntryInput(key, val string, secret *bool) *gqlschema.ConfigEntryInput {
	return &gqlschema.ConfigEntryInput{
		Key:    key,
		Value:  val,
		Secret: secret,
	}
}
