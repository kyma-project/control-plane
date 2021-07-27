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
	kymaVersionWithoutTiller                   = "1.15"
	clusterEssentialsComponent                 = "cluster-essentials"
	coreComponent                              = "core"
	rafterComponent                            = "rafter"
	applicationConnectorComponent              = "application-connector"
	rafterSourceURL                            = "github.com/kyma-project/kyma.git//resources/rafter"
	gardenerProject                            = "gardener-project"
	defaultEnableKubernetesVersionAutoUpdate   = false
	defaultEnableMachineImageVersionAutoUpdate = false
	forceAllowPrivilegedContainers             = false
)

func Test_ProvisioningInputToCluster(t *testing.T) {

	releaseProvider := &realeaseMocks.Provider{}
	releaseProvider.On("GetReleaseByVersion", kymaVersion).Return(fixKymaRelease(), nil)
	releaseProvider.On("GetReleaseByVersion", kymaVersionWithoutTiller).Return(fixKymaReleaseWithoutTiller(), nil)

	gcpGardenerProvider := &gqlschema.GCPProviderConfigInput{Zones: []string{"fix-gcp-zone-1", "fix-gcp-zone-2"}}

	modelProductionProfile := model.ProductionProfile
	gqlProductionProfile := gqlschema.KymaProfileProduction

	modelEvaluationProfile := model.EvaluationProfile
	gqlEvaluationProfile := gqlschema.KymaProfileEvaluation

	gardenerGCPGQLInput := gqlschema.ProvisionRuntimeInput{
		RuntimeInput: &gqlschema.RuntimeInput{
			Name:        "runtimeName",
			Description: nil,
			Labels:      &gqlschema.Labels{},
		},
		ClusterConfig: &gqlschema.ClusterConfigInput{
			GardenerConfig: &gqlschema.GardenerConfigInput{
				Name:                              "verylon",
				KubernetesVersion:                 "version",
				VolumeSizeGb:                      util.IntPtr(1024),
				MachineType:                       "n1-standard-1",
				Region:                            "region",
				Provider:                          "GCP",
				Purpose:                           util.StringPtr("testing"),
				Seed:                              util.StringPtr("gcp-eu1"),
				TargetSecret:                      "secret",
				DiskType:                          util.StringPtr("ssd"),
				WorkerCidr:                        "cidr",
				AutoScalerMin:                     1,
				AutoScalerMax:                     5,
				MaxSurge:                          1,
				MaxUnavailable:                    2,
				EnableKubernetesVersionAutoUpdate: util.BoolPtr(true),
				ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
					GcpConfig: gcpGardenerProvider,
				},
				OidcConfig:        oidcInput(),
				ExposureClassName: util.StringPtr("internet"),
			},
			Administrators: []string{administrator},
		},
		KymaConfig: fixKymaGraphQLConfigInput(&gqlProductionProfile),
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
			VolumeSizeGB:                        util.IntPtr(1024),
			DiskType:                            util.StringPtr("ssd"),
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
			AllowPrivilegedContainers:           true,
			GardenerProviderConfig:              expectedGCPProviderCfg,
			OIDCConfig:                          oidcConfig(),
			ExposureClassName:                   util.StringPtr("internet"),
		},
		Kubeconfig:     nil,
		KymaConfig:     fixKymaConfig(&modelProductionProfile),
		Tenant:         tenant,
		SubAccountId:   util.StringPtr(subAccountId),
		Administrators: []string{administrator},
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
					Name:                              "verylon",
					KubernetesVersion:                 "version",
					VolumeSizeGb:                      util.IntPtr(1024),
					MachineType:                       "n1-standard-1",
					MachineImage:                      util.StringPtr("gardenlinux"),
					MachineImageVersion:               util.StringPtr("25.0.0"),
					Region:                            "region",
					Provider:                          "Azure",
					Purpose:                           util.StringPtr("testing"),
					TargetSecret:                      "secret",
					DiskType:                          util.StringPtr("ssd"),
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
					OidcConfig:        oidcInput(),
					ExposureClassName: util.StringPtr("internet"),
				},
				Administrators: []string{administrator},
			},
			KymaConfig: fixKymaGraphQLConfigInput(&gqlProductionProfile),
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
				VolumeSizeGB:                        util.IntPtr(1024),
				DiskType:                            util.StringPtr("ssd"),
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
				AllowPrivilegedContainers:           true,
				GardenerProviderConfig:              expectedAzureProviderCfg,
				OIDCConfig:                          oidcConfig(),
				ExposureClassName:                   util.StringPtr("internet"),
			},
			Kubeconfig:     nil,
			KymaConfig:     fixKymaConfig(&modelProductionProfile),
			Tenant:         tenant,
			SubAccountId:   util.StringPtr(subAccountId),
			Administrators: []string{administrator},
		}
	}

	gardenerAzureGQLInputWithoutTiller := createGQLRuntimeInputAzure(nil)
	gardenerAzureGQLInputWithoutTiller.KymaConfig.Version = kymaVersionWithoutTiller
	expectedGardenerAzureRuntimeConfigWithoutTiller := expectedGardenerAzureRuntimeConfig(nil)
	expectedGardenerAzureRuntimeConfigWithoutTiller.ClusterConfig.AllowPrivilegedContainers = false
	expectedGardenerAzureRuntimeConfigWithoutTiller.KymaConfig.Release = fixKymaReleaseWithoutTiller()

	gardenerAzureGQLInputWithNoTillerButAllowedPrivilegedContainers := createGQLRuntimeInputAzure(nil)
	gardenerAzureGQLInputWithNoTillerButAllowedPrivilegedContainers.ClusterConfig.GardenerConfig.AllowPrivilegedContainers = util.BoolPtr(true)
	gardenerAzureGQLInputWithNoTillerButAllowedPrivilegedContainers.KymaConfig.Version = kymaVersionWithoutTiller
	expectedGardenerAzureRuntimeConfigWithNoTillerButAllowedPrivilegedContainers := expectedGardenerAzureRuntimeConfig(nil)
	expectedGardenerAzureRuntimeConfigWithNoTillerButAllowedPrivilegedContainers.ClusterConfig.AllowPrivilegedContainers = true
	expectedGardenerAzureRuntimeConfigWithNoTillerButAllowedPrivilegedContainers.KymaConfig.Release = fixKymaReleaseWithoutTiller()

	awsGardenerProvider := &gqlschema.AWSProviderConfigInput{
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

	gardenerAWSGQLInput := gqlschema.ProvisionRuntimeInput{
		RuntimeInput: &gqlschema.RuntimeInput{
			Name:        "runtimeName",
			Description: nil,
			Labels:      &gqlschema.Labels{},
		},
		ClusterConfig: &gqlschema.ClusterConfigInput{
			GardenerConfig: &gqlschema.GardenerConfigInput{
				Name:                              "verylon",
				KubernetesVersion:                 "version",
				VolumeSizeGb:                      util.IntPtr(1024),
				MachineType:                       "n1-standard-1",
				Region:                            "region",
				Provider:                          "AWS",
				Purpose:                           util.StringPtr("testing"),
				Seed:                              util.StringPtr("aws-eu1"),
				TargetSecret:                      "secret",
				DiskType:                          util.StringPtr("ssd"),
				WorkerCidr:                        "cidr",
				AutoScalerMin:                     1,
				AutoScalerMax:                     5,
				MaxSurge:                          1,
				MaxUnavailable:                    2,
				EnableKubernetesVersionAutoUpdate: util.BoolPtr(true),
				ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
					AwsConfig: awsGardenerProvider,
				},
				OidcConfig: oidcInput(),
			},
			Administrators: []string{administrator},
		},
		KymaConfig: fixKymaGraphQLConfigInput(&gqlEvaluationProfile),
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
			VolumeSizeGB:                        util.IntPtr(1024),
			DiskType:                            util.StringPtr("ssd"),
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
			AllowPrivilegedContainers:           true,
			GardenerProviderConfig:              expectedAWSProviderCfg,
			OIDCConfig:                          oidcConfig(),
			ExposureClassName:                   util.StringPtr("internet"),
		},
		Kubeconfig:     nil,
		KymaConfig:     fixKymaConfig(&modelEvaluationProfile),
		Tenant:         tenant,
		SubAccountId:   util.StringPtr(subAccountId),
		Administrators: []string{administrator},
	}

	openstackGardenerProvider := &gqlschema.OpenStackProviderConfigInput{
		Zones:                []string{"eu-de-1a"},
		FloatingPoolName:     "FloatingIP-external-cp",
		CloudProfileName:     "converged-cloud-cp",
		LoadBalancerProvider: "f5",
	}

	gardenerOpenstackGQLInput := gqlschema.ProvisionRuntimeInput{
		RuntimeInput: &gqlschema.RuntimeInput{
			Name:        "runtimeName",
			Description: nil,
			Labels:      &gqlschema.Labels{},
		},
		ClusterConfig: &gqlschema.ClusterConfigInput{
			GardenerConfig: &gqlschema.GardenerConfigInput{
				Name:                              "verylon",
				KubernetesVersion:                 "version",
				MachineType:                       "large.1n",
				Region:                            "region",
				Provider:                          "Openstack",
				Purpose:                           util.StringPtr("testing"),
				Seed:                              util.StringPtr("ops-1"),
				TargetSecret:                      "secret",
				WorkerCidr:                        "cidr",
				AutoScalerMin:                     1,
				AutoScalerMax:                     5,
				MaxSurge:                          1,
				MaxUnavailable:                    2,
				EnableKubernetesVersionAutoUpdate: util.BoolPtr(true),
				ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
					OpenStackConfig: openstackGardenerProvider,
				},
				OidcConfig:        oidcInput(),
				ExposureClassName: util.StringPtr("internet"),
			},
			Administrators: []string{administrator},
		},
		KymaConfig: fixKymaGraphQLConfigInput(&gqlEvaluationProfile),
	}

	expectedOpenStackProviderCfg, err := model.NewOpenStackGardenerConfig(openstackGardenerProvider)
	require.NoError(t, err)

	expectedGardenerOpenStackRuntimeConfig := model.Cluster{
		ID: "runtimeID",
		ClusterConfig: model.GardenerConfig{
			ID:                                  "id",
			Name:                                "verylon",
			ProjectName:                         gardenerProject,
			MachineType:                         "large.1n",
			Region:                              "region",
			KubernetesVersion:                   "version",
			Provider:                            "Openstack",
			Purpose:                             util.StringPtr("testing"),
			Seed:                                "ops-1",
			TargetSecret:                        "secret",
			WorkerCidr:                          "cidr",
			AutoScalerMin:                       1,
			AutoScalerMax:                       5,
			MaxSurge:                            1,
			MaxUnavailable:                      2,
			ClusterID:                           "runtimeID",
			EnableKubernetesVersionAutoUpdate:   true,
			EnableMachineImageVersionAutoUpdate: false,
			AllowPrivilegedContainers:           true,
			GardenerProviderConfig:              expectedOpenStackProviderCfg,
			OIDCConfig:                          oidcConfig(),
			ExposureClassName:                   util.StringPtr("internet"),
		},
		Kubeconfig:     nil,
		KymaConfig:     fixKymaConfig(&modelEvaluationProfile),
		Tenant:         tenant,
		SubAccountId:   util.StringPtr(subAccountId),
		Administrators: []string{administrator},
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
			input:       gardenerAzureGQLInputWithoutTiller,
			expected:    expectedGardenerAzureRuntimeConfigWithoutTiller,
			description: "Should not allow privileged containers if Tiller is not present",
		},
		{
			input:       gardenerAzureGQLInputWithNoTillerButAllowedPrivilegedContainers,
			expected:    expectedGardenerAzureRuntimeConfigWithNoTillerButAllowedPrivilegedContainers,
			description: "Should allow privileged containers if requested even when Tiller is not present",
		},
		{
			input:       gardenerAWSGQLInput,
			expected:    expectedGardenerAWSRuntimeConfig,
			description: "Should create proper runtime config struct with Gardener input for AWS provider",
		},
		{
			input:       gardenerOpenstackGQLInput,
			expected:    expectedGardenerOpenStackRuntimeConfig,
			description: "Should create proper runtime config struct with Gardener input for OpenStack provider",
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
				releaseProvider,
				gardenerProject,
				defaultEnableKubernetesVersionAutoUpdate,
				defaultEnableMachineImageVersionAutoUpdate,
				forceAllowPrivilegedContainers)

			//when
			runtimeConfig, err := inputConverter.ProvisioningInputToCluster("runtimeID", testCase.input, tenant, subAccountId)

			//then
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, runtimeConfig)
			uuidGeneratorMock.AssertExpectations(t)
		})
	}

	t.Run("Should use force allow privileged containers if equals true even if everything else says false", func(t *testing.T) {
		// given
		gardenerAzureGQLInput := createGQLRuntimeInputAzure(nil)
		gardenerAzureGQLInput.KymaConfig.Version = kymaVersionWithoutTiller
		gardenerAzureGQLInput.ClusterConfig.GardenerConfig.AllowPrivilegedContainers = util.BoolPtr(false)

		expectedGardenerAzureRuntimeConfig := expectedGardenerAzureRuntimeConfig(nil)
		expectedGardenerAzureRuntimeConfig.KymaConfig.Release = fixKymaReleaseWithoutTiller()
		expectedGardenerAzureRuntimeConfig.ClusterConfig.AllowPrivilegedContainers = true

		uuidGeneratorMock := &mocks.UUIDGenerator{}
		uuidGeneratorMock.On("New").Return("id").Times(6)
		uuidGeneratorMock.On("New").Return("very-Long-ID-That-Has-More-Than-Fourteen-Characters-And-Even-Some-Hyphens")

		forceAllowPrivilegedContainers := true

		inputConverter := NewInputConverter(
			uuidGeneratorMock,
			releaseProvider,
			gardenerProject,
			defaultEnableKubernetesVersionAutoUpdate,
			defaultEnableMachineImageVersionAutoUpdate,
			forceAllowPrivilegedContainers)

		// when
		runtimeConfig, err := inputConverter.ProvisioningInputToCluster("runtimeID", gardenerAzureGQLInput, tenant, subAccountId)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedGardenerAzureRuntimeConfig, runtimeConfig)
		uuidGeneratorMock.AssertExpectations(t)
	})
}

func oidcInput() *gqlschema.OIDCConfigInput {
	return &gqlschema.OIDCConfigInput{
		ClientID:       "9bd05ed7-a930-44e6-8c79-e6defeb1111",
		GroupsClaim:    "groups",
		IssuerURL:      "https://kymatest.accounts400.ondemand.com",
		SigningAlgs:    []string{"RS256"},
		UsernameClaim:  "sub",
		UsernamePrefix: "-",
	}
}

func upgradedOidcInput() *gqlschema.OIDCConfigInput {
	return &gqlschema.OIDCConfigInput{
		ClientID:       "9bd05ed7-a930-44e6-8c79-e6defeb2222",
		GroupsClaim:    "groups",
		IssuerURL:      "https://kymatest.accounts400.ondemand.com",
		SigningAlgs:    []string{"RS257"},
		UsernameClaim:  "sup",
		UsernamePrefix: "-",
	}
}

func oidcConfig() *model.OIDCConfig {
	return &model.OIDCConfig{
		ClientID:       "9bd05ed7-a930-44e6-8c79-e6defeb1111",
		GroupsClaim:    "groups",
		IssuerURL:      "https://kymatest.accounts400.ondemand.com",
		SigningAlgs:    []string{"RS256"},
		UsernameClaim:  "sub",
		UsernamePrefix: "-",
	}
}

func upgradedOidcConfig() *model.OIDCConfig {
	return &model.OIDCConfig{
		ClientID:       "9bd05ed7-a930-44e6-8c79-e6defeb2222",
		GroupsClaim:    "groups",
		IssuerURL:      "https://kymatest.accounts400.ondemand.com",
		SigningAlgs:    []string{"RS257"},
		UsernameClaim:  "sup",
		UsernamePrefix: "-",
	}
}

func TestConverter_ParseInput(t *testing.T) {
	t.Run("should parse KymaConfig input", func(t *testing.T) {

		//given
		uuidGeneratorMock := &mocks.UUIDGenerator{}
		uuidGeneratorMock.On("New").Return("id").Times(6)
		uuidGeneratorMock.On("New").Return("very-Long-ID-That-Has-More-Than-Fourteen-Characters-And-Even-Some-Hyphens")

		releaseProvider := &realeaseMocks.Provider{}
		releaseProvider.On("GetReleaseByVersion", kymaVersion).Return(fixKymaRelease(), nil)
		releaseProvider.On("GetReleaseByVersion", kymaVersionWithoutTiller).Return(fixKymaReleaseWithoutTiller(), nil)

		replace := gqlschema.ConflictStrategyReplace
		input := gqlschema.KymaConfigInput{
			Version:          kymaVersion,
			ConflictStrategy: &replace,
		}

		inputConverter := NewInputConverter(
			uuidGeneratorMock,
			releaseProvider,
			gardenerProject,
			defaultEnableKubernetesVersionAutoUpdate,
			defaultEnableMachineImageVersionAutoUpdate,
			forceAllowPrivilegedContainers)

		// when
		output, err := inputConverter.KymaConfigFromInput("runtimeID", input)

		// then
		require.NoError(t, err)
		assert.Equal(t, gqlschema.ConflictStrategyReplace.String(), output.GlobalConfiguration.ConflictStrategy)
		for _, entry := range output.Components {
			assert.Equal(t, gqlschema.ConflictStrategyReplace.String(), entry.Configuration.ConflictStrategy)
		}
	})
}

func TestConverter_ProvisioningInputToCluster_Error(t *testing.T) {

	t.Run("should return error when failed to get kyma release", func(t *testing.T) {
		// given
		uuidGeneratorMock := &mocks.UUIDGenerator{}
		releaseProvider := &realeaseMocks.Provider{}
		releaseProvider.On("GetReleaseByVersion", kymaVersion).Return(model.Release{}, dberrors.NotFound("error"))

		input := gqlschema.ProvisionRuntimeInput{
			ClusterConfig: &gqlschema.ClusterConfigInput{
				GardenerConfig: &gqlschema.GardenerConfigInput{},
				Administrators: []string{administrator},
			},
			KymaConfig: &gqlschema.KymaConfigInput{
				Version: kymaVersion,
			},
		}

		inputConverter := NewInputConverter(
			uuidGeneratorMock,
			releaseProvider,
			gardenerProject,
			defaultEnableKubernetesVersionAutoUpdate,
			defaultEnableMachineImageVersionAutoUpdate,
			forceAllowPrivilegedContainers)

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
			defaultEnableMachineImageVersionAutoUpdate,
			forceAllowPrivilegedContainers)

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
				Administrators: []string{administrator},
			},
		}

		inputConverter := NewInputConverter(
			nil,
			nil,
			gardenerProject,
			defaultEnableKubernetesVersionAutoUpdate,
			defaultEnableMachineImageVersionAutoUpdate,
			forceAllowPrivilegedContainers)

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
				Administrators: []string{administrator},
			},
		}

		inputConverter := NewInputConverter(
			uuidGeneratorMock,
			nil,
			gardenerProject,
			defaultEnableKubernetesVersionAutoUpdate,
			defaultEnableMachineImageVersionAutoUpdate,
			forceAllowPrivilegedContainers)

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

	releaseProvider := &realeaseMocks.Provider{}
	releaseProvider.On("GetReleaseByVersion", kymaVersion).Return(fixKymaRelease(), nil)

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
				VolumeSizeGB:           util.IntPtr(1),
				DiskType:               util.StringPtr("ssd"),
				MachineType:            "1",
				Purpose:                &evaluationPurpose,
				AutoScalerMin:          1,
				AutoScalerMax:          2,
				MaxSurge:               1,
				MaxUnavailable:         1,
				GardenerProviderConfig: initialGCPProviderConfig,
				OIDCConfig:             oidcConfig(),
				ExposureClassName:      util.StringPtr("internet"),
			},
			upgradedConfig: model.GardenerConfig{
				KubernetesVersion:      "version2",
				VolumeSizeGB:           util.IntPtr(50),
				DiskType:               util.StringPtr("papyrus"),
				MachineType:            "new-machine",
				Purpose:                &testingPurpose,
				AutoScalerMin:          2,
				AutoScalerMax:          6,
				MaxSurge:               2,
				MaxUnavailable:         1,
				GardenerProviderConfig: upgradedGCPProviderConfig,
				OIDCConfig:             upgradedOidcConfig(),
				ExposureClassName:      util.StringPtr("internet"),
			},
		},
		{description: "regular Azure shoot upgrade",
			upgradeInput: newAzureUpgradeShootInput(testingPurpose),
			initialConfig: model.GardenerConfig{
				KubernetesVersion:      "version",
				VolumeSizeGB:           util.IntPtr(1),
				DiskType:               util.StringPtr("ssd"),
				MachineType:            "1",
				Purpose:                &evaluationPurpose,
				AutoScalerMin:          1,
				AutoScalerMax:          2,
				MaxSurge:               1,
				MaxUnavailable:         1,
				GardenerProviderConfig: initialAzureProviderConfig,
				OIDCConfig:             oidcConfig(),
				ExposureClassName:      util.StringPtr("internet"),
			},
			upgradedConfig: model.GardenerConfig{
				KubernetesVersion:      "version2",
				VolumeSizeGB:           util.IntPtr(50),
				DiskType:               util.StringPtr("papyrus"),
				MachineType:            "new-machine",
				Purpose:                &testingPurpose,
				AutoScalerMin:          2,
				AutoScalerMax:          6,
				MaxSurge:               2,
				MaxUnavailable:         1,
				GardenerProviderConfig: upgradedAzureProviderConfig,
				OIDCConfig:             upgradedOidcConfig(),
				ExposureClassName:      util.StringPtr("internet"),
			},
		},
		{description: "regular AWS shoot upgrade",
			upgradeInput: newUpgradeShootInputAwsAzureGCP(testingPurpose),
			initialConfig: model.GardenerConfig{
				KubernetesVersion: "version",
				VolumeSizeGB:      util.IntPtr(1),
				DiskType:          util.StringPtr("ssd"),
				MachineType:       "1",
				Purpose:           &evaluationPurpose,
				AutoScalerMin:     1,
				AutoScalerMax:     2,
				MaxSurge:          1,
				MaxUnavailable:    1,
				OIDCConfig:        oidcConfig(),
				ExposureClassName: util.StringPtr("internet"),
			},
			upgradedConfig: model.GardenerConfig{
				KubernetesVersion: "version2",
				VolumeSizeGB:      util.IntPtr(50),
				DiskType:          util.StringPtr("papyrus"),
				MachineType:       "new-machine",
				Purpose:           &testingPurpose,
				AutoScalerMin:     2,
				AutoScalerMax:     6,
				MaxSurge:          2,
				MaxUnavailable:    1,
				OIDCConfig:        upgradedOidcConfig(),
				ExposureClassName: util.StringPtr("internet"),
			},
		},
		{description: "regular OpenStack shoot upgrade",
			upgradeInput: newUpgradeOpenStackShootInput(testingPurpose),
			initialConfig: model.GardenerConfig{
				KubernetesVersion: "version",
				MachineType:       "1",
				Purpose:           &evaluationPurpose,
				AutoScalerMin:     1,
				AutoScalerMax:     2,
				MaxSurge:          1,
				MaxUnavailable:    1,
				OIDCConfig:        oidcConfig(),
				ExposureClassName: util.StringPtr("internet"),
			},
			upgradedConfig: model.GardenerConfig{
				KubernetesVersion: "version2",
				MachineType:       "new-machine",
				Purpose:           &testingPurpose,
				AutoScalerMin:     2,
				AutoScalerMax:     6,
				MaxSurge:          2,
				MaxUnavailable:    1,
				OIDCConfig:        upgradedOidcConfig(),
				ExposureClassName: util.StringPtr("internet"),
			},
		},
		{description: "shoot upgrade with nil values",
			upgradeInput: newUpgradeShootInputWithNilValues(),
			initialConfig: model.GardenerConfig{
				KubernetesVersion: "version",
				VolumeSizeGB:      util.IntPtr(1),
				DiskType:          util.StringPtr("ssd"),
				MachineType:       "1",
				Purpose:           &evaluationPurpose,
				AutoScalerMin:     1,
				AutoScalerMax:     2,
				MaxSurge:          1,
				MaxUnavailable:    1,
				OIDCConfig:        oidcConfig(),
				ExposureClassName: util.StringPtr("internet"),
			},
			upgradedConfig: model.GardenerConfig{
				KubernetesVersion: "version",
				VolumeSizeGB:      util.IntPtr(1),
				DiskType:          util.StringPtr("ssd"),
				MachineType:       "1",
				Purpose:           &evaluationPurpose,
				AutoScalerMin:     1,
				AutoScalerMax:     2,
				MaxSurge:          1,
				MaxUnavailable:    1,
				OIDCConfig:        upgradedOidcConfig(),
				ExposureClassName: util.StringPtr("internet"),
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
				VolumeSizeGB:           util.IntPtr(1),
				DiskType:               util.StringPtr("ssd"),
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
				releaseProvider,
				gardenerProject,
				defaultEnableKubernetesVersionAutoUpdate,
				defaultEnableMachineImageVersionAutoUpdate,
				forceAllowPrivilegedContainers,
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
				releaseProvider,
				gardenerProject,
				defaultEnableKubernetesVersionAutoUpdate,
				defaultEnableMachineImageVersionAutoUpdate,
				forceAllowPrivilegedContainers,
			)

			//when
			_, err := inputConverter.UpgradeShootInputToGardenerConfig(*testCase.upgradeInput.GardenerConfig, testCase.initialConfig)

			//then
			require.Error(t, err)
			uuidGeneratorMock.AssertExpectations(t)
		})
	}
}

func newUpgradeShootInputAwsAzureGCP(newPurpose string) gqlschema.UpgradeShootInput {
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
			OidcConfig:             upgradedOidcInput(),
		},
		Administrators: []string{"test@test.pl"},
	}
}

func newUpgradeOpenStackShootInput(newPurpose string) gqlschema.UpgradeShootInput {
	return gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion:      util.StringPtr("version2"),
			Purpose:                &newPurpose,
			MachineType:            util.StringPtr("new-machine"),
			AutoScalerMin:          util.IntPtr(2),
			AutoScalerMax:          util.IntPtr(6),
			MaxSurge:               util.IntPtr(2),
			MaxUnavailable:         util.IntPtr(1),
			ProviderSpecificConfig: nil,
			OidcConfig:             upgradedOidcInput(),
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
			OidcConfig:             upgradedOidcInput(),
		},
	}
}

func newGCPUpgradeShootInput(newPurpose string) gqlschema.UpgradeShootInput {
	input := newUpgradeShootInputAwsAzureGCP(newPurpose)
	input.GardenerConfig.ProviderSpecificConfig = &gqlschema.ProviderSpecificInput{
		GcpConfig: &gqlschema.GCPProviderConfigInput{
			Zones: []string{"europe-west1-a", "europe-west1-b"},
		},
	}
	return input
}

func newAzureUpgradeShootInput(newPurpose string) gqlschema.UpgradeShootInput {
	input := newUpgradeShootInputAwsAzureGCP(newPurpose)
	input.GardenerConfig.ProviderSpecificConfig = &gqlschema.ProviderSpecificInput{
		AzureConfig: &gqlschema.AzureProviderConfigInput{
			Zones: []string{"1", "2"},
		},
	}
	return input
}

func newUpgradeShootInputWithoutProviderConfig(newPurpose string) gqlschema.UpgradeShootInput {
	input := newUpgradeShootInputAwsAzureGCP(newPurpose)
	input.GardenerConfig.ProviderSpecificConfig = &gqlschema.ProviderSpecificInput{
		AwsConfig:   nil,
		AzureConfig: nil,
		GcpConfig:   nil,
	}
	return input
}

func fixKymaGraphQLConfigInput(profile *gqlschema.KymaProfile) *gqlschema.KymaConfigInput {
	return &gqlschema.KymaConfigInput{
		Version: kymaVersion,
		Profile: profile,
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

func fixKymaReleaseWithoutTiller() model.Release {
	return model.Release{
		Id:            "e829b1b5-2e82-426d-91b0-f94978c0c140",
		Version:       kymaVersionWithoutTiller,
		TillerYAML:    "",
		InstallerYAML: "installer yaml",
	}
}
