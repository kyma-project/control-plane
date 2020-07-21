package testkit

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
)

func CreateGardenerProvisioningInput(config *TestConfig, version, provider string) (gqlschema.ProvisionRuntimeInput, error) {
	gardenerInputs := map[string]gqlschema.GardenerConfigInput{
		GCP: {
			MachineType:  "n1-standard-4",
			DiskType:     "pd-standard",
			Region:       "europe-west4",
			TargetSecret: config.Gardener.GCPSecret,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				GcpConfig: &gqlschema.GCPProviderConfigInput{
					Zones: []string{"europe-west4-a", "europe-west4-b", "europe-west4-c"},
				},
			},
		},
		Azure: {
			MachineType:  "Standard_D4_v3",
			DiskType:     "Standard_LRS",
			Region:       "westeurope",
			TargetSecret: config.Gardener.AzureSecret,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AzureConfig: &gqlschema.AzureProviderConfigInput{
					VnetCidr: "10.250.0.0/19",
					Zones:    []string{"1", "2", "3"},
				},
			},
		},
	}

	kymaConfigInput, err := CreateKymaConfigInput(version)
	if err != nil {
		return gqlschema.ProvisionRuntimeInput{}, fmt.Errorf("failed to create kyma config input: %s", err.Error())
	}

	return gqlschema.ProvisionRuntimeInput{
		RuntimeInput: &gqlschema.RuntimeInput{
			Name: "",
		},
		ClusterConfig: &gqlschema.ClusterConfigInput{
			GardenerConfig: &gqlschema.GardenerConfigInput{
				KubernetesVersion:                   "1.15.10",
				DiskType:                            gardenerInputs[provider].DiskType,
				VolumeSizeGb:                        35,
				MachineType:                         gardenerInputs[provider].MachineType,
				Region:                              gardenerInputs[provider].Region,
				Purpose:                             strToPtr("testing"),
				Provider:                            toLowerCase(provider),
				TargetSecret:                        gardenerInputs[provider].TargetSecret,
				WorkerCidr:                          "10.250.0.0/19",
				AutoScalerMin:                       2,
				AutoScalerMax:                       4,
				MaxSurge:                            4,
				MaxUnavailable:                      1,
				EnableKubernetesVersionAutoUpdate:   boolToPtr(true),
				EnableMachineImageVersionAutoUpdate: boolToPtr(false),
				ProviderSpecificConfig:              gardenerInputs[provider].ProviderSpecificConfig,
			},
		},
		KymaConfig: kymaConfigInput,
	}, nil
}

func CreateGardenerUpgradeInput(config *TestConfig, provider string) *gqlschema.UpgradeShootInput {
	return &gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion:                   strToPtr("1.15.11"),
			DiskType:                            strToPtr("Premium_LRS"),
			VolumeSizeGb:                        intToPtr(50),
			MachineType:                         strToPtr("Standard_D8_v3"),
			Purpose:                             strToPtr("evaluation"),
			AutoScalerMin:                       intToPtr(1),
			AutoScalerMax:                       intToPtr(5),
			MaxSurge:                            intToPtr(5),
			MaxUnavailable:                      intToPtr(2),
			EnableKubernetesVersionAutoUpdate:   boolToPtr(false),
			EnableMachineImageVersionAutoUpdate: boolToPtr(true),
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AzureConfig: &gqlschema.AzureProviderConfigInput{
					VnetCidr: "10.250.0.0/19",
					Zones:    []string{"1", "2", "3"},
				},
			},
		},
	}
}

func CreateKymaConfigInput(version string) (*gqlschema.KymaConfigInput, error) {
	installationCRURL := createInstallationCRURL(version)
	logrus.Infof("Getting and parsing Kyma modules from Installation CR at: %s", installationCRURL)
	componentConfigInput, err := GetAndParseInstallerCR(installationCRURL)
	if err != nil {
		return &gqlschema.KymaConfigInput{}, fmt.Errorf("failed to create component config input: %s", err.Error())
	}

	return &gqlschema.KymaConfigInput{Version: version, Components: componentConfigInput}, nil
}
