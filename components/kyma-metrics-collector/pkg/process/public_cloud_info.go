package process

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/env"
)

type Providers struct {
	Azure AzureMachines
	AWS   AWSMachines
}

type AzureMachines map[string]Feature

type AWSMachines map[string]Feature

type Feature struct {
	CpuCores int     `json:"cpu_cores"`
	Memory   float64 `json:"memory"`
	Storage  int     `json:"storage,omitempty"`
	MaxNICs  int     `json:"max_nics,omitempty"`
}

type MachineInfo map[string]json.RawMessage

func (p Providers) GetFeature(cloudProvider, vmType string) (f *Feature) {
	switch cloudProvider {
	case AWS:
		if feature, ok := p.AWS[vmType]; ok {
			return &feature
		}
	case Azure:
		if feature, ok := p.Azure[vmType]; ok {
			return &feature
		}
	}
	return nil
}

// LoadPublicCloudSpecs loads string data to Providers object from an env var
func LoadPublicCloudSpecs(cfg *env.Config) (*Providers, error) {
	if cfg.PublicCloudSpecs == "" {
		return nil, fmt.Errorf("public cloud specification is not configured")
	}

	var machineInfo MachineInfo
	err := json.Unmarshal([]byte(cfg.PublicCloudSpecs), &machineInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal machine info")
	}
	awsMachinesData, err := machineInfo[AWS].MarshalJSON()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal AWS info")
	}
	awsMachines := &AWSMachines{}
	err = json.Unmarshal(awsMachinesData, &awsMachines)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal AWS machines data")
	}
	azureMachinesData, err := machineInfo[Azure].MarshalJSON()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal Azure info")
	}
	azureMachines := &AzureMachines{}
	err = json.Unmarshal(azureMachinesData, azureMachines)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to uunmarshal Azure machines data")
	}

	providers := Providers{
		AWS:   *awsMachines,
		Azure: *azureMachines,
	}

	return &providers, nil
}
