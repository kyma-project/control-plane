package provisioning

import (
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/util"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/assert"
)

const (
	kymaSystemNamespace      = "kyma-system"
	kymaIntegrationNamespace = "kyma-integration"
)

func TestOperationStatusToGQLOperationStatus(t *testing.T) {

	graphQLConverter := NewGraphQLConverter()

	t.Run("Should create proper operation status struct", func(t *testing.T) {
		//given
		operation := model.Operation{
			ID:        "5f6e3ab6-d803-430a-8fac-29c9c9b4485a",
			Type:      model.Upgrade,
			State:     model.InProgress,
			Message:   "Some message",
			ClusterID: "6af76034-272a-42be-ac39-30e075f515a3",
		}

		operationID := "5f6e3ab6-d803-430a-8fac-29c9c9b4485a"
		message := "Some message"
		runtimeID := "6af76034-272a-42be-ac39-30e075f515a3"

		expectedOperationStatus := &gqlschema.OperationStatus{
			ID:        &operationID,
			Operation: gqlschema.OperationTypeUpgrade,
			State:     gqlschema.OperationStateInProgress,
			Message:   &message,
			RuntimeID: &runtimeID,
		}

		//when
		status := graphQLConverter.OperationStatusToGQLOperationStatus(operation)

		//then
		assert.Equal(t, expectedOperationStatus, status)
	})
}

func TestRuntimeStatusToGraphQLStatus(t *testing.T) {

	graphQLConverter := NewGraphQLConverter()

	t.Run("Should create proper runtime status struct for gardener config with zones", func(t *testing.T) {
		//given
		clusterName := "Something"
		project := "Project"
		disk := "standard"
		machine := "machine"
		machineImage := "gardenlinux"
		machineImageVersion := "25.0.0"
		region := "region"
		zones := []string{"fix-gcp-zone-1", "fix-gcp-zone-2"}
		volume := 256
		kubeversion := "kubeversion"
		kubeconfig := "kubeconfig"
		provider := "GCP"
		purpose := "testing"
		licenceType := "partner"
		seed := "gcp-eu1"
		secret := "secret"
		cidr := "cidr"
		autoScMax := 2
		autoScMin := 2
		surge := 1
		unavailable := 1
		enableKubernetesVersionAutoUpdate := true
		enableMachineImageVersionAutoUpdate := false
		allowPrivilegedContainers := true
		exposureClassName := "internet"

		gardenerProviderConfig, err := model.NewGardenerProviderConfigFromJSON(`{"zones":["fix-gcp-zone-1","fix-gcp-zone-2"]}`)
		require.NoError(t, err)

		runtimeStatus := model.RuntimeStatus{
			LastOperationStatus: model.Operation{
				ID:        "5f6e3ab6-d803-430a-8fac-29c9c9b4485a",
				Type:      model.Deprovision,
				State:     model.Failed,
				Message:   "Some message",
				ClusterID: "6af76034-272a-42be-ac39-30e075f515a3",
			},
			RuntimeConnectionStatus: model.RuntimeAgentConnectionStatusDisconnected,
			RuntimeConfiguration: model.Cluster{
				ClusterConfig: model.GardenerConfig{
					Name:                                clusterName,
					ProjectName:                         project,
					DiskType:                            &disk,
					MachineType:                         machine,
					MachineImage:                        &machineImage,
					MachineImageVersion:                 &machineImageVersion,
					Region:                              region,
					VolumeSizeGB:                        &volume,
					KubernetesVersion:                   kubeversion,
					Provider:                            provider,
					Purpose:                             &purpose,
					LicenceType:                         &licenceType,
					Seed:                                seed,
					TargetSecret:                        secret,
					WorkerCidr:                          cidr,
					AutoScalerMax:                       autoScMax,
					AutoScalerMin:                       autoScMin,
					MaxSurge:                            surge,
					MaxUnavailable:                      unavailable,
					EnableKubernetesVersionAutoUpdate:   enableKubernetesVersionAutoUpdate,
					EnableMachineImageVersionAutoUpdate: enableMachineImageVersionAutoUpdate,
					AllowPrivilegedContainers:           allowPrivilegedContainers,
					GardenerProviderConfig:              gardenerProviderConfig,
					OIDCConfig:                          oidcConfig(),
					ExposureClassName:                   &exposureClassName,
				},
				Kubeconfig: &kubeconfig,
				KymaConfig: fixKymaConfig(nil),
			},
			HibernationStatus: model.HibernationStatus{
				HibernationPossible: true,
				Hibernated:          true,
			},
		}

		operationID := "5f6e3ab6-d803-430a-8fac-29c9c9b4485a"
		message := "Some message"
		runtimeID := "6af76034-272a-42be-ac39-30e075f515a3"

		hibernationPossible := true
		hibernated := true

		expectedRuntimeStatus := &gqlschema.RuntimeStatus{
			LastOperationStatus: &gqlschema.OperationStatus{
				ID:        &operationID,
				Operation: gqlschema.OperationTypeDeprovision,
				State:     gqlschema.OperationStateFailed,
				Message:   &message,
				RuntimeID: &runtimeID,
			},
			RuntimeConnectionStatus: &gqlschema.RuntimeConnectionStatus{
				Status: gqlschema.RuntimeAgentConnectionStatusDisconnected,
			},
			RuntimeConfiguration: &gqlschema.RuntimeConfig{
				ClusterConfig: &gqlschema.GardenerConfig{
					Name:                                &clusterName,
					DiskType:                            &disk,
					MachineType:                         &machine,
					MachineImage:                        &machineImage,
					MachineImageVersion:                 &machineImageVersion,
					Region:                              &region,
					VolumeSizeGb:                        &volume,
					KubernetesVersion:                   &kubeversion,
					Provider:                            &provider,
					Purpose:                             &purpose,
					LicenceType:                         &licenceType,
					Seed:                                &seed,
					TargetSecret:                        &secret,
					WorkerCidr:                          &cidr,
					AutoScalerMax:                       &autoScMax,
					AutoScalerMin:                       &autoScMin,
					MaxSurge:                            &surge,
					MaxUnavailable:                      &unavailable,
					EnableKubernetesVersionAutoUpdate:   &enableKubernetesVersionAutoUpdate,
					EnableMachineImageVersionAutoUpdate: &enableMachineImageVersionAutoUpdate,
					AllowPrivilegedContainers:           &allowPrivilegedContainers,
					ProviderSpecificConfig: gqlschema.GCPProviderConfig{
						Zones: zones,
					},
					OidcConfig: &gqlschema.OIDCConfig{
						ClientID:       "9bd05ed7-a930-44e6-8c79-e6defeb1111",
						GroupsClaim:    "groups",
						IssuerURL:      "https://kymatest.accounts400.ondemand.com",
						SigningAlgs:    []string{"RS256"},
						UsernameClaim:  "sub",
						UsernamePrefix: "-",
					},
					ExposureClassName: &exposureClassName,
				},
				KymaConfig: fixKymaGraphQLConfig(nil),
				Kubeconfig: &kubeconfig,
			},
			HibernationStatus: &gqlschema.HibernationStatus{
				HibernationPossible: &hibernationPossible,
				Hibernated:          &hibernated,
			},
		}

		//when
		gqlStatus := graphQLConverter.RuntimeStatusToGraphQLStatus(runtimeStatus)

		//then
		assert.Equal(t, expectedRuntimeStatus, gqlStatus)
	})

	t.Run("Should create proper runtime status struct for gardener config without kyma config", func(t *testing.T) {
		//given
		clusterName := "Something"
		project := "Project"
		disk := "standard"
		machine := "machine"
		machineImage := "gardenlinux"
		machineImageVersion := "25.0.0"
		region := "region"
		zones := []string{"fix-gcp-zone-1", "fix-gcp-zone-2"}
		volume := 256
		kubeversion := "kubeversion"
		kubeconfig := "kubeconfig"
		provider := "GCP"
		purpose := "testing"
		licenceType := "partner"
		seed := "gcp-eu1"
		secret := "secret"
		cidr := "cidr"
		autoScMax := 2
		autoScMin := 2
		surge := 1
		unavailable := 1
		enableKubernetesVersionAutoUpdate := true
		enableMachineImageVersionAutoUpdate := false
		allowPrivilegedContainers := true
		exposureClassName := "internet"

		gardenerProviderConfig, err := model.NewGardenerProviderConfigFromJSON(`{"zones":["fix-gcp-zone-1","fix-gcp-zone-2"]}`)
		require.NoError(t, err)

		runtimeStatus := model.RuntimeStatus{
			LastOperationStatus: model.Operation{
				ID:        "5f6e3ab6-d803-430a-8fac-29c9c9b4485a",
				Type:      model.DeprovisionNoInstall,
				State:     model.Failed,
				Message:   "Some message",
				ClusterID: "6af76034-272a-42be-ac39-30e075f515a3",
			},
			RuntimeConnectionStatus: model.RuntimeAgentConnectionStatusDisconnected,
			RuntimeConfiguration: model.Cluster{
				ClusterConfig: model.GardenerConfig{
					Name:                                clusterName,
					ProjectName:                         project,
					DiskType:                            &disk,
					MachineType:                         machine,
					MachineImage:                        &machineImage,
					MachineImageVersion:                 &machineImageVersion,
					Region:                              region,
					VolumeSizeGB:                        &volume,
					KubernetesVersion:                   kubeversion,
					Provider:                            provider,
					Purpose:                             &purpose,
					LicenceType:                         &licenceType,
					Seed:                                seed,
					TargetSecret:                        secret,
					WorkerCidr:                          cidr,
					AutoScalerMax:                       autoScMax,
					AutoScalerMin:                       autoScMin,
					MaxSurge:                            surge,
					MaxUnavailable:                      unavailable,
					EnableKubernetesVersionAutoUpdate:   enableKubernetesVersionAutoUpdate,
					EnableMachineImageVersionAutoUpdate: enableMachineImageVersionAutoUpdate,
					AllowPrivilegedContainers:           allowPrivilegedContainers,
					GardenerProviderConfig:              gardenerProviderConfig,
					OIDCConfig:                          oidcConfig(),
					ExposureClassName:                   &exposureClassName,
				},
				Kubeconfig: &kubeconfig,
			},
			HibernationStatus: model.HibernationStatus{
				HibernationPossible: true,
				Hibernated:          true,
			},
		}

		operationID := "5f6e3ab6-d803-430a-8fac-29c9c9b4485a"
		message := "Some message"
		runtimeID := "6af76034-272a-42be-ac39-30e075f515a3"

		hibernationPossible := true
		hibernated := true

		expectedRuntimeStatus := &gqlschema.RuntimeStatus{
			LastOperationStatus: &gqlschema.OperationStatus{
				ID:        &operationID,
				Operation: gqlschema.OperationTypeDeprovisionNoInstall,
				State:     gqlschema.OperationStateFailed,
				Message:   &message,
				RuntimeID: &runtimeID,
			},
			RuntimeConnectionStatus: &gqlschema.RuntimeConnectionStatus{
				Status: gqlschema.RuntimeAgentConnectionStatusDisconnected,
			},
			RuntimeConfiguration: &gqlschema.RuntimeConfig{
				ClusterConfig: &gqlschema.GardenerConfig{
					Name:                                &clusterName,
					DiskType:                            &disk,
					MachineType:                         &machine,
					MachineImage:                        &machineImage,
					MachineImageVersion:                 &machineImageVersion,
					Region:                              &region,
					VolumeSizeGb:                        &volume,
					KubernetesVersion:                   &kubeversion,
					Provider:                            &provider,
					Purpose:                             &purpose,
					LicenceType:                         &licenceType,
					Seed:                                &seed,
					TargetSecret:                        &secret,
					WorkerCidr:                          &cidr,
					AutoScalerMax:                       &autoScMax,
					AutoScalerMin:                       &autoScMin,
					MaxSurge:                            &surge,
					MaxUnavailable:                      &unavailable,
					EnableKubernetesVersionAutoUpdate:   &enableKubernetesVersionAutoUpdate,
					EnableMachineImageVersionAutoUpdate: &enableMachineImageVersionAutoUpdate,
					AllowPrivilegedContainers:           &allowPrivilegedContainers,
					ProviderSpecificConfig: gqlschema.GCPProviderConfig{
						Zones: zones,
					},
					OidcConfig: &gqlschema.OIDCConfig{
						ClientID:       "9bd05ed7-a930-44e6-8c79-e6defeb1111",
						GroupsClaim:    "groups",
						IssuerURL:      "https://kymatest.accounts400.ondemand.com",
						SigningAlgs:    []string{"RS256"},
						UsernameClaim:  "sub",
						UsernamePrefix: "-",
					},
					ExposureClassName: &exposureClassName,
				},
				Kubeconfig: &kubeconfig,
			},
			HibernationStatus: &gqlschema.HibernationStatus{
				HibernationPossible: &hibernationPossible,
				Hibernated:          &hibernated,
			},
		}

		//when
		gqlStatus := graphQLConverter.RuntimeStatusToGraphQLStatus(runtimeStatus)

		//then
		assert.Equal(t, expectedRuntimeStatus, gqlStatus)
	})

	t.Run("Should create proper runtime status struct for gardener config without zones", func(t *testing.T) {
		//given
		clusterName := "Something"
		project := "Project"
		disk := "standard"
		machine := "machine"
		machineImage := "gardenlinux"
		machineImageVersion := "25.0.0"
		region := "region"
		volume := 256
		kubeversion := "kubeversion"
		kubeconfig := "kubeconfig"
		provider := "Azure"
		purpose := "testing"
		licenceType := ""
		seed := "az-eu1"
		secret := "secret"
		cidr := "cidr"
		autoScMax := 2
		autoScMin := 2
		surge := 1
		unavailable := 1
		enableKubernetesVersionAutoUpdate := true
		enableMachineImageVersionAutoUpdate := false
		allowPrivilegedContainers := true

		modelProductionProfile := model.ProductionProfile
		gqlProductionProfile := gqlschema.KymaProfileProduction

		gardenerProviderConfig, err := model.NewGardenerProviderConfigFromJSON(`{"vnetCidr":"10.10.11.11/255"}`)
		require.NoError(t, err)

		runtimeStatus := model.RuntimeStatus{
			LastOperationStatus: model.Operation{
				ID:        "5f6e3ab6-d803-430a-8fac-29c9c9b4485a",
				Type:      model.Deprovision,
				State:     model.Failed,
				Message:   "Some message",
				ClusterID: "6af76034-272a-42be-ac39-30e075f515a3",
			},
			RuntimeConnectionStatus: model.RuntimeAgentConnectionStatusDisconnected,
			RuntimeConfiguration: model.Cluster{
				ClusterConfig: model.GardenerConfig{
					Name:                                clusterName,
					ProjectName:                         project,
					KubernetesVersion:                   kubeversion,
					VolumeSizeGB:                        &volume,
					DiskType:                            &disk,
					MachineType:                         machine,
					MachineImage:                        &machineImage,
					MachineImageVersion:                 &machineImageVersion,
					Provider:                            provider,
					Purpose:                             &purpose,
					LicenceType:                         &licenceType,
					Seed:                                seed,
					TargetSecret:                        secret,
					Region:                              region,
					WorkerCidr:                          cidr,
					AutoScalerMin:                       autoScMin,
					AutoScalerMax:                       autoScMax,
					MaxSurge:                            surge,
					MaxUnavailable:                      unavailable,
					EnableKubernetesVersionAutoUpdate:   enableKubernetesVersionAutoUpdate,
					EnableMachineImageVersionAutoUpdate: enableMachineImageVersionAutoUpdate,
					AllowPrivilegedContainers:           allowPrivilegedContainers,
					GardenerProviderConfig:              gardenerProviderConfig,
				},
				Kubeconfig: &kubeconfig,
				KymaConfig: fixKymaConfig(&modelProductionProfile),
			},
			HibernationStatus: model.HibernationStatus{
				HibernationPossible: true,
				Hibernated:          true,
			},
		}

		operationID := "5f6e3ab6-d803-430a-8fac-29c9c9b4485a"
		message := "Some message"
		runtimeID := "6af76034-272a-42be-ac39-30e075f515a3"
		hibernationPossible := true
		hibernated := true

		expectedRuntimeStatus := &gqlschema.RuntimeStatus{
			LastOperationStatus: &gqlschema.OperationStatus{
				ID:        &operationID,
				Operation: gqlschema.OperationTypeDeprovision,
				State:     gqlschema.OperationStateFailed,
				Message:   &message,
				RuntimeID: &runtimeID,
			},
			RuntimeConnectionStatus: &gqlschema.RuntimeConnectionStatus{
				Status: gqlschema.RuntimeAgentConnectionStatusDisconnected,
			},
			RuntimeConfiguration: &gqlschema.RuntimeConfig{
				ClusterConfig: &gqlschema.GardenerConfig{
					Name:                                &clusterName,
					DiskType:                            &disk,
					MachineType:                         &machine,
					MachineImage:                        &machineImage,
					MachineImageVersion:                 &machineImageVersion,
					Region:                              &region,
					VolumeSizeGb:                        &volume,
					KubernetesVersion:                   &kubeversion,
					Provider:                            &provider,
					Purpose:                             &purpose,
					LicenceType:                         &licenceType,
					Seed:                                &seed,
					TargetSecret:                        &secret,
					WorkerCidr:                          &cidr,
					AutoScalerMax:                       &autoScMax,
					AutoScalerMin:                       &autoScMin,
					MaxSurge:                            &surge,
					MaxUnavailable:                      &unavailable,
					EnableKubernetesVersionAutoUpdate:   &enableKubernetesVersionAutoUpdate,
					EnableMachineImageVersionAutoUpdate: &enableMachineImageVersionAutoUpdate,
					AllowPrivilegedContainers:           &allowPrivilegedContainers,
					ProviderSpecificConfig: gqlschema.AzureProviderConfig{
						VnetCidr: util.StringPtr("10.10.11.11/255"),
						Zones:    nil, // Expected empty when no zones specified in input.
					},
				},
				KymaConfig: fixKymaGraphQLConfig(&gqlProductionProfile),
				Kubeconfig: &kubeconfig,
			},
			HibernationStatus: &gqlschema.HibernationStatus{
				HibernationPossible: &hibernationPossible,
				Hibernated:          &hibernated,
			},
		}

		//when
		gqlStatus := graphQLConverter.RuntimeStatusToGraphQLStatus(runtimeStatus)

		//then
		assert.Equal(t, expectedRuntimeStatus, gqlStatus)
	})
}

func fixKymaGraphQLConfig(profile *gqlschema.KymaProfile) *gqlschema.KymaConfig {
	return &gqlschema.KymaConfig{
		Version: util.StringPtr(kymaVersion),
		Profile: profile,
		Components: []*gqlschema.ComponentConfiguration{
			{
				Component:     clusterEssentialsComponent,
				Namespace:     kymaSystemNamespace,
				Configuration: make([]*gqlschema.ConfigEntry, 0, 0),
			},
			{
				Component: coreComponent,
				Namespace: kymaSystemNamespace,
				Configuration: []*gqlschema.ConfigEntry{
					fixGQLConfigEntry("test.config.key", "value", util.BoolPtr(false)),
					fixGQLConfigEntry("test.config.key2", "value2", util.BoolPtr(false)),
				},
			},
			{
				Component:     rafterComponent,
				Namespace:     kymaSystemNamespace,
				SourceURL:     util.StringPtr(rafterSourceURL),
				Configuration: make([]*gqlschema.ConfigEntry, 0, 0),
			},
			{
				Component: applicationConnectorComponent,
				Namespace: kymaIntegrationNamespace,
				Configuration: []*gqlschema.ConfigEntry{
					fixGQLConfigEntry("test.config.key", "value", util.BoolPtr(false)),
					fixGQLConfigEntry("test.secret.key", "secretValue", util.BoolPtr(true)),
				},
			},
		},
		Configuration: []*gqlschema.ConfigEntry{
			fixGQLConfigEntry("global.config.key", "globalValue", util.BoolPtr(false)),
			fixGQLConfigEntry("global.config.key2", "globalValue2", util.BoolPtr(false)),
			fixGQLConfigEntry("global.secret.key", "globalSecretValue", util.BoolPtr(true)),
		},
	}
}

func fixGQLConfigEntry(key, val string, secret *bool) *gqlschema.ConfigEntry {
	return &gqlschema.ConfigEntry{
		Key:    key,
		Value:  val,
		Secret: secret,
	}
}

func fixKymaConfig(profile *model.KymaProfile) *model.KymaConfig {
	return &model.KymaConfig{
		ID:                  "id",
		Release:             fixKymaRelease(),
		Profile:             profile,
		Components:          fixKymaComponents(),
		GlobalConfiguration: fixGlobalConfig(),
		ClusterID:           "runtimeID",
	}
}

func fixGlobalConfig() model.Configuration {
	return model.Configuration{
		ConfigEntries: []model.ConfigEntry{
			model.NewConfigEntry("global.config.key", "globalValue", false),
			model.NewConfigEntry("global.config.key2", "globalValue2", false),
			model.NewConfigEntry("global.secret.key", "globalSecretValue", true),
		},
	}
}

func fixKymaComponents() []model.KymaComponentConfig {
	return []model.KymaComponentConfig{
		{
			ID:             "id",
			KymaConfigID:   "id",
			Component:      clusterEssentialsComponent,
			Namespace:      kymaSystemNamespace,
			Configuration:  model.Configuration{ConfigEntries: make([]model.ConfigEntry, 0, 0)},
			ComponentOrder: 1,
		},
		{
			ID:           "id",
			KymaConfigID: "id",
			Component:    coreComponent,
			Namespace:    kymaSystemNamespace,
			Configuration: model.Configuration{
				ConfigEntries: []model.ConfigEntry{
					model.NewConfigEntry("test.config.key", "value", false),
					model.NewConfigEntry("test.config.key2", "value2", false),
				},
			},
			ComponentOrder: 2,
		},
		{
			ID:             "id",
			KymaConfigID:   "id",
			Component:      rafterComponent,
			Namespace:      kymaSystemNamespace,
			SourceURL:      util.StringPtr(rafterSourceURL),
			Configuration:  model.Configuration{ConfigEntries: make([]model.ConfigEntry, 0, 0)},
			ComponentOrder: 3,
		},
		{
			ID:           "id",
			KymaConfigID: "id",
			Component:    applicationConnectorComponent,
			Namespace:    kymaIntegrationNamespace,
			Configuration: model.Configuration{
				ConfigEntries: []model.ConfigEntry{
					model.NewConfigEntry("test.config.key", "value", false),
					model.NewConfigEntry("test.secret.key", "secretValue", true),
				},
			},
			ComponentOrder: 4,
		},
	}
}

func fixKymaRelease() model.Release {
	return model.Release{
		Id:            "d829b1b5-2e82-426d-91b0-f94978c0c140",
		Version:       kymaVersion,
		TillerYAML:    "tiller yaml",
		InstallerYAML: "installer yaml",
	}
}
