package testkit

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/google/uuid"

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
				Name:                                strToPtr(createGardenerClusterName()),
				AllowPrivilegedContainers:           boolToPtr(true),
				KubernetesVersion:                   config.KubernetesVersion,
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

func CreateGardenerUpgradeInput(config *TestConfig) *gqlschema.UpgradeShootInput {
	return &gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion:                   strToPtr(config.UpgradeKubernetesVersion),
			DiskType:                            strToPtr("Standard_LRS"),
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
	componentConfigInput := fixKymaComponentList()

	return &gqlschema.KymaConfigInput{Version: version, Components: componentConfigInput}, nil
}

func fixKymaComponentList() []*gqlschema.ComponentConfigurationInput {
	clusterEssentials := &gqlschema.ComponentConfigurationInput{
		Component: "cluster-essentials",
		Namespace: "kyma-system",
	}

	testing := &gqlschema.ComponentConfigurationInput{
		Component: "testing",
		Namespace: "kyma-system",
	}

	istio := &gqlschema.ComponentConfigurationInput{
		Component: "istio",
		Namespace: "istio-system",
	}

	xipPatch := &gqlschema.ComponentConfigurationInput{
		Component: "xip-patch",
		Namespace: "kyma-installer",
	}

	core := &gqlschema.ComponentConfigurationInput{
		Component: "core",
		Namespace: "kyma-system",
	}

	applicationConnector := &gqlschema.ComponentConfigurationInput{
		Component: "application-connector",
		Namespace: "kyma-system",
	}

	runtimeAgent := &gqlschema.ComponentConfigurationInput{
		Component: "compass-runtime-agent",
		Namespace: "kyma-system",
		Configuration: []*gqlschema.ConfigEntryInput{
			{
				Key:    "global.disableLegacyConnectivity",
				Value:  "true",
				Secret: boolToPtr(false),
			},
		},
	}

	return []*gqlschema.ComponentConfigurationInput{
		clusterEssentials,
		testing,
		istio,
		xipPatch,
		core,
		applicationConnector,
		runtimeAgent,
	}
}

func createGardenerClusterName() string {
	id := uuid.New().String()

	name := strings.ReplaceAll(id, "-", "")
	name = fmt.Sprintf("%.7s", name)
	name = startWithLetter(name)
	name = strings.ToLower(name)
	return name
}

func startWithLetter(str string) string {
	if len(str) == 0 {
		return "c"
	} else if !unicode.IsLetter(rune(str[0])) {
		return fmt.Sprintf("c-%.9s", str)
	}
	return str
}
