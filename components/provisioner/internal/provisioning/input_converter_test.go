package provisioning

import (
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/util"

	realeaseMocks "github.com/kyma-project/control-plane/components/provisioner/internal/installation/release/mocks"

	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"

	"github.com/kyma-project/control-plane/components/provisioner/internal/uuid/mocks"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/assert"
)

const (
	kymaVersion                                = "1.5"
	clusterEssentialsComponent                 = "cluster-essentials"
	coreComponent                              = "core"
	rafterComponent                            = "rafter"
	applicationConnectorComponent              = "application-connector"
	rafterSourceURL                            = "github.com/kyma-project/kyma.git//resources/rafter"
	gardenerProject                            = "gardener-project"
	defaultEnableKubernetesVersionAutoUpdate   = false
	defaultEnableMachineImageVersionAutoUpdate = false
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
				KubernetesVersion:                 "version",
				VolumeSizeGb:                      1024,
				MachineType:                       "n1-standard-1",
				Region:                            "region",
				Provider:                          "GCP",
				Purpose:                           util.StringPtr("testing"),
				Seed:                              util.StringPtr("gcp-eu1"),
				TargetSecret:                      "secret",
				DiskType:                          "ssd",
				WorkerCidr:                        "cidr",
				AutoScalerMin:                     1,
				AutoScalerMax:                     5,
				MaxSurge:                          1,
				MaxUnavailable:                    2,
				EnableKubernetesVersionAutoUpdate: util.BoolPtr(true),
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
			ID:                                  "id",
			Name:                                "verylon",
			ProjectName:                         gardenerProject,
			MachineType:                         "n1-standard-1",
			Region:                              "region",
			KubernetesVersion:                   "version",
			VolumeSizeGB:                        1024,
			DiskType:                            "ssd",
			Provider:                            "GCP",
			Purpose:                             util.StringPtr("testing"),
			Seed:                                "gcp-eu1",
			TargetSecret:                        "secret",
			WorkerCidr:                          "cidr",
			AutoScalerMin:                       1,
			AutoScalerMax:                       5,
			MaxSurge:                            1,
			MaxUnavailable:                      2,
			ClusterID:                           "runtimeID",
			EnableKubernetesVersionAutoUpdate:   true,
			EnableMachineImageVersionAutoUpdate: false,
			GardenerProviderConfig:              expectedGCPProviderCfg,
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
					KubernetesVersion:                 "version",
					VolumeSizeGb:                      1024,
					MachineType:                       "n1-standard-1",
					MachineImage:                      util.StringPtr("gardenlinux"),
					MachineImageVersion:               util.StringPtr("25.0.0"),
					Region:                            "region",
					Provider:                          "Azure",
					Purpose:                           util.StringPtr("testing"),
					TargetSecret:                      "secret",
					DiskType:                          "ssd",
					WorkerCidr:                        "cidr",
					AutoScalerMin:                     1,
					AutoScalerMax:                     5,
					MaxSurge:                          1,
					MaxUnavailable:                    2,
					EnableKubernetesVersionAutoUpdate: util.BoolPtr(true),
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
				ID:                                  "id",
				Name:                                "verylon",
				ProjectName:                         gardenerProject,
				MachineType:                         "n1-standard-1",
				MachineImage:                        util.StringPtr("gardenlinux"),
				MachineImageVersion:                 util.StringPtr("25.0.0"),
				Region:                              "region",
				KubernetesVersion:                   "version",
				VolumeSizeGB:                        1024,
				DiskType:                            "ssd",
				Provider:                            "Azure",
				Purpose:                             util.StringPtr("testing"),
				Seed:                                "",
				TargetSecret:                        "secret",
				WorkerCidr:                          "cidr",
				AutoScalerMin:                       1,
				AutoScalerMax:                       5,
				MaxSurge:                            1,
				MaxUnavailable:                      2,
				ClusterID:                           "runtimeID",
				EnableKubernetesVersionAutoUpdate:   true,
				EnableMachineImageVersionAutoUpdate: false,
				GardenerProviderConfig:              expectedAzureProviderCfg,
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
				KubernetesVersion:                 "version",
				VolumeSizeGb:                      1024,
				MachineType:                       "n1-standard-1",
				Region:                            "region",
				Provider:                          "AWS",
				Purpose:                           util.StringPtr("testing"),
				Seed:                              util.StringPtr("aws-eu1"),
				TargetSecret:                      "secret",
				DiskType:                          "ssd",
				WorkerCidr:                        "cidr",
				AutoScalerMin:                     1,
				AutoScalerMax:                     5,
				MaxSurge:                          1,
				MaxUnavailable:                    2,
				EnableKubernetesVersionAutoUpdate: util.BoolPtr(true),
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
			ID:                                  "id",
			Name:                                "verylon",
			ProjectName:                         gardenerProject,
			MachineType:                         "n1-standard-1",
			Region:                              "region",
			KubernetesVersion:                   "version",
			VolumeSizeGB:                        1024,
			DiskType:                            "ssd",
			Provider:                            "AWS",
			Purpose:                             util.StringPtr("testing"),
			Seed:                                "aws-eu1",
			TargetSecret:                        "secret",
			WorkerCidr:                          "cidr",
			AutoScalerMin:                       1,
			AutoScalerMax:                       5,
			MaxSurge:                            1,
			MaxUnavailable:                      2,
			ClusterID:                           "runtimeID",
			EnableKubernetesVersionAutoUpdate:   true,
			EnableMachineImageVersionAutoUpdate: false,
			GardenerProviderConfig:              expectedAWSProviderCfg,
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

			inputConverter := NewInputConverter(
				uuidGeneratorMock,
				readSession,
				gardenerProject,
				defaultEnableKubernetesVersionAutoUpdate,
				defaultEnableMachineImageVersionAutoUpdate)

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

		inputConverter := NewInputConverter(
			uuidGeneratorMock,
			readSession,
			gardenerProject,
			defaultEnableKubernetesVersionAutoUpdate,
			defaultEnableMachineImageVersionAutoUpdate)

		//when
		_, err := inputConverter.ProvisioningInputToCluster("runtimeID", input, tenant, subAccountId)

		//then
		require.Error(t, err)
		uuidGeneratorMock.AssertExpectations(t)
	})

	t.Run("should return error when Cluster Config not provided", func(t *testing.T) {
		// given
		input := gqlschema.ProvisionRuntimeInput{}

		inputConverter := NewInputConverter(
			nil,
			nil,
			gardenerProject,
			defaultEnableKubernetesVersionAutoUpdate,
			defaultEnableMachineImageVersionAutoUpdate)

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

		inputConverter := NewInputConverter(
			nil,
			nil,
			gardenerProject,
			defaultEnableKubernetesVersionAutoUpdate,
			defaultEnableMachineImageVersionAutoUpdate)

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

		inputConverter := NewInputConverter(
			uuidGeneratorMock,
			nil,
			gardenerProject,
			defaultEnableKubernetesVersionAutoUpdate,
			defaultEnableMachineImageVersionAutoUpdate)

		//when
		_, err := inputConverter.ProvisioningInputToCluster("runtimeID", input, tenant, subAccountId)

		//then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider config not specified")
	})

}

func Test_UpgradeShootInputToGardenerConfig(t *testing.T) {
	evaluationPurpose := "evaluation"
	testingPurpose := "testing"

	readSession := &realeaseMocks.Repository{}
	readSession.On("GetReleaseByVersion", kymaVersion).Return(fixKymaRelease(), nil)

	initialGCPProviderConfig, _ := model.NewGCPGardenerConfig(&gqlschema.GCPProviderConfigInput{Zones: []string{"europe-west1-a"}})
	upgradedGCPProviderConfig, _ := model.NewGCPGardenerConfig(&gqlschema.GCPProviderConfigInput{Zones: []string{"europe-west1-a", "europe-west1-b"}})

	initialAzureProviderConfig, _ := model.NewAzureGardenerConfig(&gqlschema.AzureProviderConfigInput{Zones: []string{"1"}})
	upgradedAzureProviderConfig, _ := model.NewAzureGardenerConfig(&gqlschema.AzureProviderConfigInput{Zones: []string{"1", "2"}})

	casesWithNoErrors := []struct {
		description    string
		upgradeInput   gqlschema.UpgradeShootInput
		initialConfig  model.GardenerConfig
		upgradedConfig model.GardenerConfig
	}{
		{description: "regular GCP shoot upgrade",
			upgradeInput: newGCPUpgradeShootInput(testingPurpose),
			initialConfig: model.GardenerConfig{
				KubernetesVersion:      "version",
				VolumeSizeGB:           1,
				DiskType:               "ssd",
				MachineType:            "1",
				Purpose:                &evaluationPurpose,
				AutoScalerMin:          1,
				AutoScalerMax:          2,
				MaxSurge:               1,
				MaxUnavailable:         1,
				GardenerProviderConfig: initialGCPProviderConfig,
			},
			upgradedConfig: model.GardenerConfig{
				KubernetesVersion:      "version2",
				VolumeSizeGB:           50,
				DiskType:               "papyrus",
				MachineType:            "new-machine",
				Purpose:                &testingPurpose,
				AutoScalerMin:          2,
				AutoScalerMax:          6,
				MaxSurge:               2,
				MaxUnavailable:         1,
				GardenerProviderConfig: upgradedGCPProviderConfig,
			},
		},
		{description: "regular Azure shoot upgrade",
			upgradeInput: newAzureUpgradeShootInput(testingPurpose),
			initialConfig: model.GardenerConfig{
				KubernetesVersion:      "version",
				VolumeSizeGB:           1,
				DiskType:               "ssd",
				MachineType:            "1",
				Purpose:                &evaluationPurpose,
				AutoScalerMin:          1,
				AutoScalerMax:          2,
				MaxSurge:               1,
				MaxUnavailable:         1,
				GardenerProviderConfig: initialAzureProviderConfig,
			},
			upgradedConfig: model.GardenerConfig{
				KubernetesVersion:      "version2",
				VolumeSizeGB:           50,
				DiskType:               "papyrus",
				MachineType:            "new-machine",
				Purpose:                &testingPurpose,
				AutoScalerMin:          2,
				AutoScalerMax:          6,
				MaxSurge:               2,
				MaxUnavailable:         1,
				GardenerProviderConfig: upgradedAzureProviderConfig,
			},
		},
		{description: "regular AWS shoot upgrade",
			upgradeInput: newUpgradeShootInput(testingPurpose),
			initialConfig: model.GardenerConfig{
				KubernetesVersion: "version",
				VolumeSizeGB:      1,
				DiskType:          "ssd",
				MachineType:       "1",
				Purpose:           &evaluationPurpose,
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
				AutoScalerMin:     2,
				AutoScalerMax:     6,
				MaxSurge:          2,
				MaxUnavailable:    1,
			},
		},
		{description: "shoot upgrade with nil values",
			upgradeInput: newUpgradeShootInputWithNilValues(),
			initialConfig: model.GardenerConfig{
				KubernetesVersion: "version",
				VolumeSizeGB:      1,
				DiskType:          "ssd",
				MachineType:       "1",
				Purpose:           &evaluationPurpose,
				AutoScalerMin:     1,
				AutoScalerMax:     2,
				MaxSurge:          1,
				MaxUnavailable:    1,
			},
			upgradedConfig: model.GardenerConfig{
				KubernetesVersion: "version",
				VolumeSizeGB:      1,
				DiskType:          "ssd",
				MachineType:       "1",
				Purpose:           &evaluationPurpose,
				AutoScalerMin:     1,
				AutoScalerMax:     2,
				MaxSurge:          1,
				MaxUnavailable:    1,
			},
		},
	}

	casesWithErrors := []struct {
		description   string
		upgradeInput  gqlschema.UpgradeShootInput
		initialConfig model.GardenerConfig
	}{
		{description: "should return error failed to convert provider specific config",
			upgradeInput: newUpgradeShootInputWithoutProviderConfig(testingPurpose),
			initialConfig: model.GardenerConfig{
				KubernetesVersion:      "version",
				VolumeSizeGB:           1,
				DiskType:               "ssd",
				MachineType:            "1",
				Purpose:                &evaluationPurpose,
				AutoScalerMin:          1,
				AutoScalerMax:          2,
				MaxSurge:               1,
				MaxUnavailable:         1,
				GardenerProviderConfig: initialGCPProviderConfig,
			},
		},
	}

	for _, testCase := range casesWithNoErrors {
		t.Run(testCase.description, func(t *testing.T) {
			//given
			uuidGeneratorMock := &mocks.UUIDGenerator{}
			inputConverter := NewInputConverter(
				uuidGeneratorMock,
				readSession,
				gardenerProject,
				defaultEnableKubernetesVersionAutoUpdate,
				defaultEnableMachineImageVersionAutoUpdate,
			)

			//when
			shootConfig, err := inputConverter.UpgradeShootInputToGardenerConfig(*testCase.upgradeInput.GardenerConfig, testCase.initialConfig)

			//then
			require.NoError(t, err)
			assert.Equal(t, testCase.upgradedConfig, shootConfig)
			uuidGeneratorMock.AssertExpectations(t)
		})
	}

	for _, testCase := range casesWithErrors {
		t.Run(testCase.description, func(t *testing.T) {
			//given
			uuidGeneratorMock := &mocks.UUIDGenerator{}
			inputConverter := NewInputConverter(
				uuidGeneratorMock,
				readSession,
				gardenerProject,
				defaultEnableKubernetesVersionAutoUpdate,
				defaultEnableMachineImageVersionAutoUpdate,
			)

			//when
			_, err := inputConverter.UpgradeShootInputToGardenerConfig(*testCase.upgradeInput.GardenerConfig, testCase.initialConfig)

			//then
			require.Error(t, err)
			uuidGeneratorMock.AssertExpectations(t)
		})
	}
}

func newUpgradeShootInput(newPurpose string) gqlschema.UpgradeShootInput {
	return gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion:      util.StringPtr("version2"),
			Purpose:                &newPurpose,
			MachineType:            util.StringPtr("new-machine"),
			DiskType:               util.StringPtr("papyrus"),
			VolumeSizeGb:           util.IntPtr(50),
			AutoScalerMin:          util.IntPtr(2),
			AutoScalerMax:          util.IntPtr(6),
			MaxSurge:               util.IntPtr(2),
			MaxUnavailable:         util.IntPtr(1),
			ProviderSpecificConfig: nil,
		},
	}
}

func newUpgradeShootInputWithNilValues() gqlschema.UpgradeShootInput {
	return gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion:      nil,
			Purpose:                nil,
			MachineType:            nil,
			DiskType:               nil,
			VolumeSizeGb:           nil,
			AutoScalerMin:          nil,
			AutoScalerMax:          nil,
			MaxSurge:               nil,
			MaxUnavailable:         nil,
			ProviderSpecificConfig: nil,
		},
	}
}

func newGCPUpgradeShootInput(newPurpose string) gqlschema.UpgradeShootInput {
	input := newUpgradeShootInput(newPurpose)
	input.GardenerConfig.ProviderSpecificConfig = &gqlschema.ProviderSpecificInput{
		GcpConfig: &gqlschema.GCPProviderConfigInput{
			Zones: []string{"europe-west1-a", "europe-west1-b"},
		},
	}
	return input
}

func newAzureUpgradeShootInput(newPurpose string) gqlschema.UpgradeShootInput {
	input := newUpgradeShootInput(newPurpose)
	input.GardenerConfig.ProviderSpecificConfig = &gqlschema.ProviderSpecificInput{
		AzureConfig: &gqlschema.AzureProviderConfigInput{
			Zones: []string{"1", "2"},
		},
	}
	return input
}

func newUpgradeShootInputWithoutProviderConfig(newPurpose string) gqlschema.UpgradeShootInput {
	input := newUpgradeShootInput(newPurpose)
	input.GardenerConfig.ProviderSpecificConfig = &gqlschema.ProviderSpecificInput{
		AwsConfig:   nil,
		AzureConfig: nil,
		GcpConfig:   nil,
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
