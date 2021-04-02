package process

import (
	"encoding/json"
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/env"
)

type Providers struct {
	Data Data `json:"data"`
}

type Data map[string]interface{}

type Vm struct {
	VmSpecs *map[string]interface{} `json:"vm_specs"`
}

type Features struct {
	*Feature `json:"features"`
}

type Feature struct {
	CpuCores int     `json:"cpu_cores"`
	Memory   float64 `json:"memory"`
	Storage  int64   `json:"storage"`
	MaxNICs  int     `json:"max_nics"`
}

func (p Providers) GetFeatures(cloudProvider, vmType string) (f *Features) {
	if providerVm, ok := p.Data[cloudProvider].(*Vm); ok {
		spec := *providerVm.VmSpecs
		if features, ok := spec[vmType].(*Features); ok {
			f = features
		}
	}
	return
}

// LoadPublicCloudSpecs loads string data to Providers object from an env var
func LoadPublicCloudSpecs(cfg *env.Config) (*Providers, error) {
	if cfg.PublicCloudSpecs == "" {
		return nil, fmt.Errorf("public cloud specification is not configured")
	}
	providers := new(Providers)
	err := json.Unmarshal([]byte(cfg.PublicCloudSpecs), providers)
	if err != nil {
		return nil, err
	}

	for provider := range providers.Data {

		vmByte, err := json.Marshal(providers.Data[provider])
		if err != nil {
			return nil, err
		}
		vm := new(Vm)
		err = json.Unmarshal(vmByte, vm)
		if err != nil {
			return nil, err
		}
		spec := *vm.VmSpecs
		for sp := range spec {
			featuresByte, err := json.Marshal(spec[sp])
			if err != nil {
				return nil, err
			}
			features := new(Features)
			err = json.Unmarshal(featuresByte, features)
			if err != nil {
				return nil, err
			}
			providers.Data[provider] = vm
			spec := *vm.VmSpecs
			spec[sp] = features
		}
	}
	return providers, nil
}
