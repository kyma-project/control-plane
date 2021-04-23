package provisioning

import (
	"fmt"
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/testkit"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/assert"
)

var (
	providerGCP   = "GCP"
	providerAzure = "Azure"

	clusterName         = "Something"
	project             = "Project"
	disk                = "standard"
	machine             = "machine"
	machineImage        = "gardenlinux"
	machineImageVersion = "25.0.0"
	region              = "region"
	volume              = 256
	kubeversion         = "kubeversion"
	kubeconfig          = "kubeconfig"
	purpose             = "testing"
	secret              = "secret"
	cidr                = "10.10.11.11/255"
	autoScMax           = 2
	autoScMin           = 2
	surge               = 1
	unavailable         = 1

	allowPrivilegedContainers           = true
	enableKubernetesVersionAutoUpdate   = true
	enableMachineImageVersionAutoUpdate = false
)

var (
	zones = []string{"fix-gcp-zone-1", "fix-gcp-zone-2"}
	seed  = map[string]string{
		providerAzure: "az-eu1",
		providerGCP:   "gcp-eu1",
	}
	licenceType = map[string]string{
		providerAzure: "",
		providerGCP:   "partner",
	}
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

		expectedOperationID := "5f6e3ab6-d803-430a-8fac-29c9c9b4485a"
		expectedMessage := "Some message"
		expectedRuntimeID := "6af76034-272a-42be-ac39-30e075f515a3"

		expectedOperationStatus := &gqlschema.OperationStatus{
			ID:        &expectedOperationID,
			Operation: gqlschema.OperationTypeUpgrade,
			State:     gqlschema.OperationStateInProgress,
			Message:   &expectedMessage,
			RuntimeID: &expectedRuntimeID,
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
		provider := providerGCP

		gardenerProviderConfig, err := model.NewGardenerProviderConfigFromJSON(fmt.Sprintf(`{"zones":["%s","%s"]}`, zones[0], zones[1]))
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
					LicenceType:                         util.StringPtr(licenceType[provider]),
					Seed:                                seed[provider],
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
				},
				Kubeconfig: &kubeconfig,
				KymaConfig: testkit.FixKymaConfig(nil),
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
					LicenceType:                         util.StringPtr(licenceType[provider]),
					Seed:                                util.StringPtr(seed[provider]),
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
				},
				KymaConfig: testkit.FixGQLKymaConfig(nil),
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
		provider := providerAzure

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
					LicenceType:                         util.StringPtr(licenceType[provider]),
					Seed:                                seed[provider],
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
				KymaConfig: testkit.FixKymaConfig(&modelProductionProfile),
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
					LicenceType:                         util.StringPtr(licenceType[provider]),
					Seed:                                util.StringPtr(seed[provider]),
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
				KymaConfig: testkit.FixGQLKymaConfig(&gqlProductionProfile),
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
