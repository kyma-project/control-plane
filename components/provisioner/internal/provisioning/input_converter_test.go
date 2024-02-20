package provisioning

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/internal/uuid/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
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
	gcpGardenerProvider := &gqlschema.GCPProviderConfigInput{Zones: []string{"fix-gcp-zone-1", "fix-gcp-zone-2"}}

	modelProductionProfile := model.ProductionProfile
	gqlProductionProfile := gqlschema.KymaProfileProduction

	modelEvaluationProfile := model.EvaluationProfile
	gqlEvaluationProfile := gqlschema.KymaProfileEvaluation

	gardenerGCPGQLInput := gqlschema.ProvisionRuntimeInput{
		RuntimeInput: &gqlschema.RuntimeInput{
			Name:        "runtimeName",
			Description: nil,
			Labels:      gqlschema.Labels{},
		},
		ClusterConfig: &gqlschema.ClusterConfigInput{
			GardenerConfig: &gqlschema.GardenerConfigInput{
				Name:                              "verylon",
				KubernetesVersion:                 "1.20.7",
				VolumeSizeGb:                      util.PtrTo(1024),
				MachineType:                       "n1-standard-1",
				MachineImage:                      util.PtrTo("gardenlinux"),
				MachineImageVersion:               util.PtrTo("25.0.0"),
				Region:                            "region",
				Provider:                          "GCP",
				Purpose:                           util.PtrTo("testing"),
				Seed:                              util.PtrTo("gcp-eu1"),
				TargetSecret:                      "secret",
				DiskType:                          util.PtrTo("ssd"),
				WorkerCidr:                        "10.254.0.0/16",
				PodsCidr:                          util.PtrTo("10.64.0.0/11"),
				ServicesCidr:                      util.PtrTo("10.243.0.0/16"),
				AutoScalerMin:                     1,
				AutoScalerMax:                     5,
				MaxSurge:                          1,
				MaxUnavailable:                    2,
				EnableKubernetesVersionAutoUpdate: util.PtrTo(true),
				ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
					GcpConfig: gcpGardenerProvider,
				},
				OidcConfig:                    oidcInput(),
				DNSConfig:                     dnsInput(),
				ExposureClassName:             util.PtrTo("internet"),
				ShootNetworkingFilterDisabled: util.PtrTo(true),
				ControlPlaneFailureTolerance:  util.PtrTo("zone"),
				EuAccess:                      util.PtrTo(true),
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
			MachineImage:                        util.PtrTo("gardenlinux"),
			MachineImageVersion:                 util.PtrTo("25.0.0"),
			Region:                              "region",
			KubernetesVersion:                   "1.20.7",
			VolumeSizeGB:                        util.PtrTo(1024),
			DiskType:                            util.PtrTo("ssd"),
			Provider:                            "GCP",
			Purpose:                             util.PtrTo("testing"),
			Seed:                                "gcp-eu1",
			TargetSecret:                        "secret",
			WorkerCidr:                          "10.254.0.0/16",
			PodsCIDR:                            util.PtrTo("10.64.0.0/11"),
			ServicesCIDR:                        util.PtrTo("10.243.0.0/16"),
			AutoScalerMin:                       1,
			AutoScalerMax:                       5,
			MaxSurge:                            1,
			MaxUnavailable:                      2,
			ClusterID:                           "runtimeID",
			EnableKubernetesVersionAutoUpdate:   true,
			EnableMachineImageVersionAutoUpdate: false,
			GardenerProviderConfig:              expectedGCPProviderCfg,
			OIDCConfig:                          oidcConfig(),
			DNSConfig:                           dnsConfig(),
			ExposureClassName:                   util.PtrTo("internet"),
			ShootNetworkingFilterDisabled:       util.PtrTo(true),
			ControlPlaneFailureTolerance:        util.PtrTo("zone"),
			EuAccess:                            true,
		},
		Kubeconfig:     nil,
		KymaConfig:     fixKymaConfig(&modelProductionProfile),
		Tenant:         tenant,
		SubAccountId:   util.PtrTo(subAccountId),
		Administrators: []string{administrator},
	}

	createGQLRuntimeInputAzure := func(zones []string) gqlschema.ProvisionRuntimeInput {
		return gqlschema.ProvisionRuntimeInput{
			RuntimeInput: &gqlschema.RuntimeInput{
				Name:        "runtimeName",
				Description: nil,
				Labels:      gqlschema.Labels{},
			},
			ClusterConfig: &gqlschema.ClusterConfigInput{
				GardenerConfig: &gqlschema.GardenerConfigInput{
					Name:                              "verylon",
					KubernetesVersion:                 "1.20.7",
					VolumeSizeGb:                      util.PtrTo(1024),
					MachineType:                       "n1-standard-1",
					MachineImage:                      util.PtrTo("gardenlinux"),
					MachineImageVersion:               util.PtrTo("25.0.0"),
					Region:                            "region",
					Provider:                          "Azure",
					Purpose:                           util.PtrTo("testing"),
					TargetSecret:                      "secret",
					DiskType:                          util.PtrTo("ssd"),
					WorkerCidr:                        "10.254.0.0/16",
					PodsCidr:                          util.PtrTo("10.64.0.0/11"),
					ServicesCidr:                      util.PtrTo("10.243.0.0/16"),
					AutoScalerMin:                     1,
					AutoScalerMax:                     5,
					MaxSurge:                          1,
					MaxUnavailable:                    2,
					EnableKubernetesVersionAutoUpdate: util.PtrTo(true),
					ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
						AzureConfig: &gqlschema.AzureProviderConfigInput{
							VnetCidr: "10.254.0.0/16",
							Zones:    zones,
						},
					},
					OidcConfig:                   oidcInput(),
					DNSConfig:                    dnsInput(),
					ExposureClassName:            util.PtrTo("internet"),
					ControlPlaneFailureTolerance: util.PtrTo("zone"),
					EuAccess:                     util.PtrTo(false),
				},
				Administrators: []string{administrator},
			},
			KymaConfig: fixKymaGraphQLConfigInput(&gqlProductionProfile),
		}
	}

	expectedGardenerAzureRuntimeConfig := func(zones []string) model.Cluster {
		expectedAzureProviderCfg, err := model.NewAzureGardenerConfig(&gqlschema.AzureProviderConfigInput{VnetCidr: "10.254.0.0/16", Zones: zones})
		require.NoError(t, err)

		return model.Cluster{
			ID: "runtimeID",
			ClusterConfig: model.GardenerConfig{
				ID:                                  "id",
				Name:                                "verylon",
				ProjectName:                         gardenerProject,
				MachineType:                         "n1-standard-1",
				MachineImage:                        util.PtrTo("gardenlinux"),
				MachineImageVersion:                 util.PtrTo("25.0.0"),
				Region:                              "region",
				KubernetesVersion:                   "1.20.7",
				VolumeSizeGB:                        util.PtrTo(1024),
				DiskType:                            util.PtrTo("ssd"),
				Provider:                            "Azure",
				Purpose:                             util.PtrTo("testing"),
				Seed:                                "",
				TargetSecret:                        "secret",
				WorkerCidr:                          "10.254.0.0/16",
				PodsCIDR:                            util.PtrTo("10.64.0.0/11"),
				ServicesCIDR:                        util.PtrTo("10.243.0.0/16"),
				AutoScalerMin:                       1,
				AutoScalerMax:                       5,
				MaxSurge:                            1,
				MaxUnavailable:                      2,
				ClusterID:                           "runtimeID",
				EnableKubernetesVersionAutoUpdate:   true,
				EnableMachineImageVersionAutoUpdate: false,
				GardenerProviderConfig:              expectedAzureProviderCfg,
				OIDCConfig:                          oidcConfig(),
				DNSConfig:                           dnsConfig(),
				ExposureClassName:                   util.PtrTo("internet"),
				ShootNetworkingFilterDisabled:       util.PtrTo(true),
				ControlPlaneFailureTolerance:        util.PtrTo("zone"),
				EuAccess:                            false,
			},
			Kubeconfig:     nil,
			KymaConfig:     fixKymaConfig(&modelProductionProfile),
			Tenant:         tenant,
			SubAccountId:   util.PtrTo(subAccountId),
			Administrators: []string{administrator},
		}
	}

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
			Labels:      gqlschema.Labels{},
		},
		ClusterConfig: &gqlschema.ClusterConfigInput{
			GardenerConfig: &gqlschema.GardenerConfigInput{
				Name:                              "verylon",
				KubernetesVersion:                 "1.20.7",
				VolumeSizeGb:                      util.PtrTo(1024),
				MachineType:                       "n1-standard-1",
				MachineImage:                      util.PtrTo("gardenlinux"),
				MachineImageVersion:               util.PtrTo("25.0.0"),
				Region:                            "region",
				Provider:                          "AWS",
				Purpose:                           util.PtrTo("testing"),
				Seed:                              util.PtrTo("aws-eu1"),
				TargetSecret:                      "secret",
				DiskType:                          util.PtrTo("ssd"),
				WorkerCidr:                        "10.254.0.0/16",
				PodsCidr:                          util.PtrTo("10.64.0.0/11"),
				ServicesCidr:                      util.PtrTo("10.243.0.0/16"),
				AutoScalerMin:                     1,
				AutoScalerMax:                     5,
				MaxSurge:                          1,
				MaxUnavailable:                    2,
				EnableKubernetesVersionAutoUpdate: util.PtrTo(true),
				ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
					AwsConfig: awsGardenerProvider,
				},
				OidcConfig:                    oidcInput(),
				DNSConfig:                     dnsInput(),
				ExposureClassName:             util.PtrTo("internet"),
				ShootNetworkingFilterDisabled: util.PtrTo(false),
				ControlPlaneFailureTolerance:  util.PtrTo("zone"),
				EuAccess:                      nil,
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
			MachineImage:                        util.PtrTo("gardenlinux"),
			MachineImageVersion:                 util.PtrTo("25.0.0"),
			Region:                              "region",
			KubernetesVersion:                   "1.20.7",
			VolumeSizeGB:                        util.PtrTo(1024),
			DiskType:                            util.PtrTo("ssd"),
			Provider:                            "AWS",
			Purpose:                             util.PtrTo("testing"),
			Seed:                                "aws-eu1",
			TargetSecret:                        "secret",
			WorkerCidr:                          "10.254.0.0/16",
			PodsCIDR:                            util.PtrTo("10.64.0.0/11"),
			ServicesCIDR:                        util.PtrTo("10.243.0.0/16"),
			AutoScalerMin:                       1,
			AutoScalerMax:                       5,
			MaxSurge:                            1,
			MaxUnavailable:                      2,
			ClusterID:                           "runtimeID",
			EnableKubernetesVersionAutoUpdate:   true,
			EnableMachineImageVersionAutoUpdate: false,
			GardenerProviderConfig:              expectedAWSProviderCfg,
			OIDCConfig:                          oidcConfig(),
			DNSConfig:                           dnsConfig(),
			ExposureClassName:                   util.PtrTo("internet"),
			ShootNetworkingFilterDisabled:       util.PtrTo(false),
			ControlPlaneFailureTolerance:        util.PtrTo("zone"),
			EuAccess:                            false,
		},
		Kubeconfig:     nil,
		KymaConfig:     fixKymaConfig(&modelEvaluationProfile),
		Tenant:         tenant,
		SubAccountId:   util.PtrTo(subAccountId),
		Administrators: []string{administrator},
	}

	openstackGardenerProvider := &gqlschema.OpenStackProviderConfigInput{
		Zones:                []string{"eu-de-1a"},
		FloatingPoolName:     util.PtrTo("FloatingIP-external-cp"),
		CloudProfileName:     "converged-cloud-cp",
		LoadBalancerProvider: "f5",
	}

	gardenerOpenstackGQLInput := gqlschema.ProvisionRuntimeInput{
		RuntimeInput: &gqlschema.RuntimeInput{
			Name:        "runtimeName",
			Description: nil,
			Labels:      gqlschema.Labels{},
		},
		ClusterConfig: &gqlschema.ClusterConfigInput{
			GardenerConfig: &gqlschema.GardenerConfigInput{
				Name:                              "verylon",
				KubernetesVersion:                 "1.20.7",
				MachineType:                       "large.1n",
				MachineImage:                      util.PtrTo("gardenlinux"),
				MachineImageVersion:               util.PtrTo("25.0.0"),
				Region:                            "region",
				Provider:                          "Openstack",
				Purpose:                           util.PtrTo("testing"),
				Seed:                              util.PtrTo("ops-1"),
				TargetSecret:                      "secret",
				WorkerCidr:                        "10.254.0.0/16",
				PodsCidr:                          util.PtrTo("10.64.0.0/11"),
				ServicesCidr:                      util.PtrTo("10.243.0.0/16"),
				AutoScalerMin:                     1,
				AutoScalerMax:                     5,
				MaxSurge:                          1,
				MaxUnavailable:                    2,
				EnableKubernetesVersionAutoUpdate: util.PtrTo(true),
				ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
					OpenStackConfig: openstackGardenerProvider,
				},
				OidcConfig:                    oidcInput(),
				DNSConfig:                     dnsInput(),
				ExposureClassName:             util.PtrTo(OpenStackExposureClassName),
				ShootNetworkingFilterDisabled: nil,
				EuAccess:                      nil,
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
			MachineImage:                        util.PtrTo("gardenlinux"),
			MachineImageVersion:                 util.PtrTo("25.0.0"),
			Region:                              "region",
			KubernetesVersion:                   "1.20.7",
			Provider:                            "Openstack",
			Purpose:                             util.PtrTo("testing"),
			Seed:                                "ops-1",
			TargetSecret:                        "secret",
			WorkerCidr:                          "10.254.0.0/16",
			PodsCIDR:                            util.PtrTo("10.64.0.0/11"),
			ServicesCIDR:                        util.PtrTo("10.243.0.0/16"),
			AutoScalerMin:                       1,
			AutoScalerMax:                       5,
			MaxSurge:                            1,
			MaxUnavailable:                      2,
			ClusterID:                           "runtimeID",
			EnableKubernetesVersionAutoUpdate:   true,
			EnableMachineImageVersionAutoUpdate: false,
			GardenerProviderConfig:              expectedOpenStackProviderCfg,
			OIDCConfig:                          oidcConfig(),
			DNSConfig:                           dnsConfig(),
			ExposureClassName:                   util.PtrTo(OpenStackExposureClassName),
			ShootNetworkingFilterDisabled:       util.PtrTo(true),
			EuAccess:                            false,
		},
		Kubeconfig:     nil,
		KymaConfig:     fixKymaConfig(&modelEvaluationProfile),
		Tenant:         tenant,
		SubAccountId:   util.PtrTo(subAccountId),
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
			// given
			uuidGeneratorMock := &mocks.UUIDGenerator{}
			uuidGeneratorMock.On("New").Return("id").Times(6)
			uuidGeneratorMock.On("New").Return("very-Long-ID-That-Has-More-Than-Fourteen-Characters-And-Even-Some-Hyphens")

			inputConverter := NewInputConverter(
				uuidGeneratorMock,
				gardenerProject,
				defaultEnableKubernetesVersionAutoUpdate,
				defaultEnableMachineImageVersionAutoUpdate)

			// when
			runtimeConfig, err := inputConverter.ProvisioningInputToCluster("runtimeID", testCase.input, tenant, subAccountId)

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, runtimeConfig)
			uuidGeneratorMock.AssertExpectations(t)
		})
	}
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

func dnsInput() *gqlschema.DNSConfigInput {
	return &gqlschema.DNSConfigInput{
		Domain: "verylon.devtest.kyma.ondemand.com",
		Providers: []*gqlschema.DNSProviderInput{
			{
				DomainsInclude: []string{"devtest.kyma.ondemand.com"},
				Primary:        true,
				SecretName:     "aws_dns_domain_secrets_test_inconverter",
				Type:           "route53_type_test",
			},
		},
	}
}

func dnsConfig() *model.DNSConfig {
	return &model.DNSConfig{
		Domain: "verylon.devtest.kyma.ondemand.com",
		Providers: []*model.DNSProvider{
			{
				DomainsInclude: []string{"verylon.devtest.kyma.ondemand.com"},
				Primary:        true,
				SecretName:     "aws_dns_domain_secrets_test_inconverter",
				Type:           "route53_type_test",
			},
		},
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
		// given
		uuidGeneratorMock := &mocks.UUIDGenerator{}
		uuidGeneratorMock.On("New").Return("id").Times(6)
		uuidGeneratorMock.On("New").Return("very-Long-ID-That-Has-More-Than-Fourteen-Characters-And-Even-Some-Hyphens")

		replace := gqlschema.ConflictStrategyReplace
		input := gqlschema.KymaConfigInput{
			Version:          kymaVersion,
			ConflictStrategy: &replace,
		}

		inputConverter := NewInputConverter(
			uuidGeneratorMock,
			gardenerProject,
			defaultEnableKubernetesVersionAutoUpdate,
			defaultEnableMachineImageVersionAutoUpdate)

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
	t.Run("should return error when Cluster Config not provided", func(t *testing.T) {
		// given
		input := gqlschema.ProvisionRuntimeInput{}

		inputConverter := NewInputConverter(
			nil,
			gardenerProject,
			defaultEnableKubernetesVersionAutoUpdate,
			defaultEnableMachineImageVersionAutoUpdate)

		// when
		_, err := inputConverter.ProvisioningInputToCluster("runtimeID", input, tenant, subAccountId)

		// then
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
			gardenerProject,
			defaultEnableKubernetesVersionAutoUpdate,
			defaultEnableMachineImageVersionAutoUpdate)

		// when
		_, err := inputConverter.ProvisioningInputToCluster("runtimeID", input, tenant, subAccountId)

		// then
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
			gardenerProject,
			defaultEnableKubernetesVersionAutoUpdate,
			defaultEnableMachineImageVersionAutoUpdate)

		// when
		_, err := inputConverter.ProvisioningInputToCluster("runtimeID", input, tenant, subAccountId)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider config not specified")
	})
}

func Test_UpgradeShootInputToGardenerConfig(t *testing.T) {
	evaluationPurpose := "evaluation"
	testingPurpose := "testing"

	initialGCPProviderConfig, _ := model.NewGCPGardenerConfig(&gqlschema.GCPProviderConfigInput{Zones: []string{"europe-west1-a"}})
	upgradedGCPProviderConfig, _ := model.NewGCPGardenerConfig(&gqlschema.GCPProviderConfigInput{Zones: []string{"europe-west1-a", "europe-west1-b"}})

	initialAzureProviderConfig, _ := model.NewAzureGardenerConfig(&gqlschema.AzureProviderConfigInput{Zones: []string{"1"}})
	upgradedAzureProviderConfig, _ := model.NewAzureGardenerConfig(&gqlschema.AzureProviderConfigInput{Zones: []string{"1", "2"}, EnableNatGateway: util.PtrTo(true)})

	casesWithNoErrors := []struct {
		description    string
		upgradeInput   gqlschema.UpgradeShootInput
		initialConfig  model.GardenerConfig
		upgradedConfig model.GardenerConfig
	}{
		{
			description:  "regular GCP shoot upgrade",
			upgradeInput: newGCPUpgradeShootInput(testingPurpose),
			initialConfig: model.GardenerConfig{
				KubernetesVersion:      "1.19",
				VolumeSizeGB:           util.PtrTo(1),
				DiskType:               util.PtrTo("ssd"),
				MachineType:            "1",
				MachineImage:           util.PtrTo("gardenlinux"),
				MachineImageVersion:    util.PtrTo("25.0.0"),
				Purpose:                &evaluationPurpose,
				AutoScalerMin:          1,
				AutoScalerMax:          2,
				MaxSurge:               1,
				MaxUnavailable:         1,
				GardenerProviderConfig: initialGCPProviderConfig,
				OIDCConfig:             oidcConfig(),
				ExposureClassName:      util.PtrTo("internet"),
			},
			upgradedConfig: model.GardenerConfig{
				KubernetesVersion:             "1.19",
				VolumeSizeGB:                  util.PtrTo(50),
				DiskType:                      util.PtrTo("papyrus"),
				MachineType:                   "new-machine",
				MachineImage:                  util.PtrTo("ubuntu"),
				MachineImageVersion:           util.PtrTo("12.0.2"),
				Purpose:                       &testingPurpose,
				AutoScalerMin:                 2,
				AutoScalerMax:                 6,
				MaxSurge:                      2,
				MaxUnavailable:                1,
				GardenerProviderConfig:        upgradedGCPProviderConfig,
				OIDCConfig:                    upgradedOidcConfig(),
				ExposureClassName:             util.PtrTo("internet"),
				ShootNetworkingFilterDisabled: util.PtrTo(true),
			},
		},
		{
			description:  "regular Azure shoot upgrade",
			upgradeInput: newAzureUpgradeShootInput(testingPurpose),
			initialConfig: model.GardenerConfig{
				KubernetesVersion:             "1.19",
				VolumeSizeGB:                  util.PtrTo(1),
				DiskType:                      util.PtrTo("ssd"),
				MachineType:                   "1",
				MachineImage:                  util.PtrTo("gardenlinux"),
				MachineImageVersion:           util.PtrTo("25.0.0"),
				Purpose:                       &evaluationPurpose,
				AutoScalerMin:                 1,
				AutoScalerMax:                 2,
				MaxSurge:                      1,
				MaxUnavailable:                1,
				GardenerProviderConfig:        initialAzureProviderConfig,
				OIDCConfig:                    oidcConfig(),
				ExposureClassName:             util.PtrTo("internet"),
				ShootNetworkingFilterDisabled: util.PtrTo(true),
			},
			upgradedConfig: model.GardenerConfig{
				KubernetesVersion:             "1.19",
				VolumeSizeGB:                  util.PtrTo(50),
				DiskType:                      util.PtrTo("papyrus"),
				MachineType:                   "new-machine",
				MachineImage:                  util.PtrTo("ubuntu"),
				MachineImageVersion:           util.PtrTo("12.0.2"),
				Purpose:                       &testingPurpose,
				AutoScalerMin:                 2,
				AutoScalerMax:                 6,
				MaxSurge:                      2,
				MaxUnavailable:                1,
				GardenerProviderConfig:        upgradedAzureProviderConfig,
				OIDCConfig:                    upgradedOidcConfig(),
				ExposureClassName:             util.PtrTo("internet"),
				ShootNetworkingFilterDisabled: util.PtrTo(true),
			},
		},
		{
			description:  "regular AWS shoot upgrade",
			upgradeInput: newUpgradeShootInputAwsAzureGCP(testingPurpose),
			initialConfig: model.GardenerConfig{
				KubernetesVersion:             "1.19",
				VolumeSizeGB:                  util.PtrTo(1),
				DiskType:                      util.PtrTo("ssd"),
				MachineType:                   "1",
				MachineImage:                  util.PtrTo("gardenlinux"),
				MachineImageVersion:           util.PtrTo("25.0.0"),
				Purpose:                       &evaluationPurpose,
				AutoScalerMin:                 1,
				AutoScalerMax:                 2,
				MaxSurge:                      1,
				MaxUnavailable:                1,
				OIDCConfig:                    oidcConfig(),
				ExposureClassName:             util.PtrTo("internet"),
				ShootNetworkingFilterDisabled: util.PtrTo(false),
			},
			upgradedConfig: model.GardenerConfig{
				KubernetesVersion:             "1.19",
				VolumeSizeGB:                  util.PtrTo(50),
				DiskType:                      util.PtrTo("papyrus"),
				MachineType:                   "new-machine",
				MachineImage:                  util.PtrTo("ubuntu"),
				MachineImageVersion:           util.PtrTo("12.0.2"),
				Purpose:                       &testingPurpose,
				AutoScalerMin:                 2,
				AutoScalerMax:                 6,
				MaxSurge:                      2,
				MaxUnavailable:                1,
				OIDCConfig:                    upgradedOidcConfig(),
				ExposureClassName:             util.PtrTo("internet"),
				ShootNetworkingFilterDisabled: util.PtrTo(true),
			},
		},
		{
			description:  "regular OpenStack shoot upgrade",
			upgradeInput: newUpgradeOpenStackShootInput(testingPurpose),
			initialConfig: model.GardenerConfig{
				KubernetesVersion:             "1.19",
				MachineType:                   "1",
				MachineImage:                  util.PtrTo("gardenlinux"),
				MachineImageVersion:           util.PtrTo("25.0.0"),
				Purpose:                       &evaluationPurpose,
				AutoScalerMin:                 1,
				AutoScalerMax:                 2,
				MaxSurge:                      1,
				MaxUnavailable:                1,
				OIDCConfig:                    oidcConfig(),
				ExposureClassName:             util.PtrTo("internet"),
				ShootNetworkingFilterDisabled: nil,
			},
			upgradedConfig: model.GardenerConfig{
				KubernetesVersion:             "1.19",
				MachineType:                   "new-machine",
				MachineImage:                  util.PtrTo("ubuntu"),
				MachineImageVersion:           util.PtrTo("12.0.2"),
				Purpose:                       &testingPurpose,
				AutoScalerMin:                 2,
				AutoScalerMax:                 6,
				MaxSurge:                      2,
				MaxUnavailable:                1,
				OIDCConfig:                    upgradedOidcConfig(),
				ExposureClassName:             util.PtrTo("internet"),
				ShootNetworkingFilterDisabled: nil,
			},
		},
		{
			description:  "shoot upgrade with nil values",
			upgradeInput: newUpgradeShootInputWithNilValues(),
			initialConfig: model.GardenerConfig{
				KubernetesVersion:             "1.20.7",
				VolumeSizeGB:                  util.PtrTo(1),
				DiskType:                      util.PtrTo("ssd"),
				MachineType:                   "1",
				MachineImage:                  util.PtrTo("gardenlinux"),
				MachineImageVersion:           util.PtrTo("25.0.0"),
				Purpose:                       &evaluationPurpose,
				AutoScalerMin:                 1,
				AutoScalerMax:                 2,
				MaxSurge:                      1,
				MaxUnavailable:                1,
				OIDCConfig:                    oidcConfig(),
				ExposureClassName:             util.PtrTo("internet"),
				ShootNetworkingFilterDisabled: util.PtrTo(false),
			},
			upgradedConfig: model.GardenerConfig{
				KubernetesVersion:             "1.20.7",
				VolumeSizeGB:                  util.PtrTo(1),
				DiskType:                      util.PtrTo("ssd"),
				MachineType:                   "1",
				MachineImage:                  util.PtrTo("gardenlinux"),
				MachineImageVersion:           util.PtrTo("25.0.0"),
				Purpose:                       &evaluationPurpose,
				AutoScalerMin:                 1,
				AutoScalerMax:                 2,
				MaxSurge:                      1,
				MaxUnavailable:                1,
				OIDCConfig:                    upgradedOidcConfig(),
				ExposureClassName:             util.PtrTo("internet"),
				ShootNetworkingFilterDisabled: util.PtrTo(false),
			},
		},
	}

	casesWithErrors := []struct {
		description   string
		upgradeInput  gqlschema.UpgradeShootInput
		initialConfig model.GardenerConfig
	}{
		{
			description:  "should return error failed to convert provider specific config",
			upgradeInput: newUpgradeShootInputWithoutProviderConfig(testingPurpose),
			initialConfig: model.GardenerConfig{
				KubernetesVersion:      "1.20.7",
				VolumeSizeGB:           util.PtrTo(1),
				DiskType:               util.PtrTo("ssd"),
				MachineType:            "1",
				MachineImage:           util.PtrTo("gardenlinux"),
				MachineImageVersion:    util.PtrTo("25.0.0"),
				Purpose:                &evaluationPurpose,
				AutoScalerMin:          1,
				AutoScalerMax:          2,
				MaxSurge:               1,
				MaxUnavailable:         1,
				GardenerProviderConfig: initialGCPProviderConfig,
				ExposureClassName:      util.PtrTo("internet"),
			},
		},
	}

	for _, testCase := range casesWithNoErrors {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			uuidGeneratorMock := &mocks.UUIDGenerator{}
			inputConverter := NewInputConverter(
				uuidGeneratorMock,
				gardenerProject,
				defaultEnableKubernetesVersionAutoUpdate,
				defaultEnableMachineImageVersionAutoUpdate,
			)

			// when
			shootConfig, err := inputConverter.UpgradeShootInputToGardenerConfig(*testCase.upgradeInput.GardenerConfig, testCase.initialConfig)

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.upgradedConfig, shootConfig)
			uuidGeneratorMock.AssertExpectations(t)
		})
	}

	for _, testCase := range casesWithErrors {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			uuidGeneratorMock := &mocks.UUIDGenerator{}
			inputConverter := NewInputConverter(
				uuidGeneratorMock,
				gardenerProject,
				defaultEnableKubernetesVersionAutoUpdate,
				defaultEnableMachineImageVersionAutoUpdate,
			)

			// when
			_, err := inputConverter.UpgradeShootInputToGardenerConfig(*testCase.upgradeInput.GardenerConfig, testCase.initialConfig)

			// then
			require.Error(t, err)
			uuidGeneratorMock.AssertExpectations(t)
		})
	}
}

func newUpgradeShootInputAwsAzureGCP(newPurpose string) gqlschema.UpgradeShootInput {
	return gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion:             util.PtrTo("1.19"),
			Purpose:                       &newPurpose,
			MachineType:                   util.PtrTo("new-machine"),
			DiskType:                      util.PtrTo("papyrus"),
			VolumeSizeGb:                  util.PtrTo(50),
			AutoScalerMin:                 util.PtrTo(2),
			AutoScalerMax:                 util.PtrTo(6),
			MaxSurge:                      util.PtrTo(2),
			MaxUnavailable:                util.PtrTo(1),
			MachineImage:                  util.PtrTo("ubuntu"),
			MachineImageVersion:           util.PtrTo("12.0.2"),
			ProviderSpecificConfig:        nil,
			OidcConfig:                    upgradedOidcInput(),
			ExposureClassName:             util.PtrTo("internet"),
			ShootNetworkingFilterDisabled: util.PtrTo(true),
		},
		Administrators: []string{"test@test.pl"},
	}
}

func newUpgradeOpenStackShootInput(newPurpose string) gqlschema.UpgradeShootInput {
	return gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion:             util.PtrTo("1.19"),
			Purpose:                       &newPurpose,
			MachineType:                   util.PtrTo("new-machine"),
			AutoScalerMin:                 util.PtrTo(2),
			AutoScalerMax:                 util.PtrTo(6),
			MaxSurge:                      util.PtrTo(2),
			MaxUnavailable:                util.PtrTo(1),
			MachineImage:                  util.PtrTo("ubuntu"),
			MachineImageVersion:           util.PtrTo("12.0.2"),
			ProviderSpecificConfig:        nil,
			OidcConfig:                    upgradedOidcInput(),
			ExposureClassName:             util.PtrTo("internet"),
			ShootNetworkingFilterDisabled: nil,
		},
	}
}

func newUpgradeShootInputWithNilValues() gqlschema.UpgradeShootInput {
	return gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion:             nil,
			Purpose:                       nil,
			MachineType:                   nil,
			DiskType:                      nil,
			MachineImage:                  nil,
			MachineImageVersion:           nil,
			VolumeSizeGb:                  nil,
			AutoScalerMin:                 nil,
			AutoScalerMax:                 nil,
			MaxSurge:                      nil,
			MaxUnavailable:                nil,
			ProviderSpecificConfig:        nil,
			OidcConfig:                    upgradedOidcInput(),
			ExposureClassName:             nil,
			ShootNetworkingFilterDisabled: nil,
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
			Zones:            []string{"1", "2"},
			EnableNatGateway: util.PtrTo(true),
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
					fixGQLConfigEntryInput("test.config.key", "value", util.PtrTo(false)),
					fixGQLConfigEntryInput("test.config.key2", "value2", util.PtrTo(false)),
				},
			},
			{
				Component: rafterComponent,
				Namespace: kymaSystemNamespace,
				SourceURL: util.PtrTo(rafterSourceURL),
			},
			{
				Component: applicationConnectorComponent,
				Namespace: kymaSystemNamespace,
				Configuration: []*gqlschema.ConfigEntryInput{
					fixGQLConfigEntryInput("test.config.key", "value", util.PtrTo(false)),
					fixGQLConfigEntryInput("test.secret.key", "secretValue", util.PtrTo(true)),
				},
			},
		},
		Configuration: []*gqlschema.ConfigEntryInput{
			fixGQLConfigEntryInput("global.config.key", "globalValue", util.PtrTo(false)),
			fixGQLConfigEntryInput("global.config.key2", "globalValue2", util.PtrTo(false)),
			fixGQLConfigEntryInput("global.secret.key", "globalSecretValue", util.PtrTo(true)),
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
