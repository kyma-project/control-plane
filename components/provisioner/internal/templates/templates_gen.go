package templates

import (
	"fmt"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"k8s.io/apimachinery/pkg/runtime"
)

const namespaceTemplate = "garden-{{ .ProjectName }}"

// TODO: missing maintenance window
// TODO: add gcp and aws as an option

func GenerateShootTemplate() ([]byte, error) {
	gardenerConfig, err := defaultGardenerConfig()
	if err != nil {
		return nil, fmt.Errorf("error when creating default GardenerConfig: %s", err.Error())
	}

	gardenerConfig = templateConfig(gardenerConfig)

	shoot, err := gardenerConfig.ToShootTemplate(namespaceTemplate, "", "")
	if err != nil {
		return nil, fmt.Errorf("error when composing Shoot: %s", err.Error())
	}

	encoder, err := defaultEncoder()
	if err != nil {
		return nil, fmt.Errorf("error when preparing encoder: %s", err.Error())
	}

	marshalled, err := runtime.Encode(encoder, shoot)
	if err != nil {
		return nil, fmt.Errorf("error when marshaling Shoot to JSON: %s", err.Error())
	}

	return marshalled, nil
}


func defaultGardenerConfig() (model.GardenerConfig, error) {
	azureConfigInput := gqlschema.AzureProviderConfigInput{
		VnetCidr: "10.250.0.0/16",
		Zones:    []string{"1", "2", "3"},
	}

	defaultAzureConifg, err := model.NewAzureGardenerConfig(&azureConfigInput)
	if err != nil {
		return model.GardenerConfig{}, fmt.Errorf("error creating default Azure config: %s", err.Error())
	}

	gardenerConfig := model.GardenerConfig{
		KubernetesVersion:                   "1.16.12",
		VolumeSizeGB:                        50,
		DiskType:                            "Standard_LRS",
		MachineType:                         "Standard_D8_v3",
		MachineImage:                        util.StringPtr("gardenlinux"),
		MachineImageVersion:                 util.StringPtr("27.1.0"),
		Provider:                            "azure",
		Purpose:                             util.StringPtr("development"),
		LicenceType:                         nil, // TODO: ??
		WorkerCidr:                          "10.250.0.0/16",
		AutoScalerMin:                       3,
		AutoScalerMax:                       10,
		MaxSurge:                            4,
		MaxUnavailable:                      1,
		EnableKubernetesVersionAutoUpdate:   false, // TODO
		EnableMachineImageVersionAutoUpdate: false, // TODO
		AllowPrivilegedContainers:           true,	// TODO: change when removing tiller
		GardenerProviderConfig:              defaultAzureConifg,
	}

	return gardenerConfig, nil
}

func templateConfig(config model.GardenerConfig) model.GardenerConfig {
	config.Name = "{{ .ShootName }}"
	config.ProjectName = "{{ .ProjectName }}"
	config.TargetSecret = "{{ .GardenerSecretName }}"
	config.Region = "{{ .Region }}"

	return config
}
