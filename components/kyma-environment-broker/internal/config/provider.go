package config

import "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"

type (
	ConfigReader interface {
		Read(kymaVersion, planName string) (string, error)
	}

	ConfigValidator interface {
		Validate(cfgString string) error
	}

	ConfigConverter interface {
		ConvertToStruct(cfgString string) (ConfigForPlan, error)
	}
)

type (
	ConfigForPlan struct {
		AdditionalComponents []runtime.KymaComponent `json:"additional-components" yaml:"additional-components"`
	}

	ConfigProvider struct {
		Reader    ConfigReader
		Validator ConfigValidator
		Converter ConfigConverter
	}
)

func (p *ConfigProvider) ProvideForGivenVersionAndPlan(kymaVersion, planName string) (*ConfigForPlan, error) {
	cfgString, err := p.Reader.Read(kymaVersion, planName)
	if err != nil {
		return nil, err
	}

	if err = p.Validator.Validate(cfgString); err != nil {
		return nil, err
	}

	cfg, err := p.Converter.ConvertToStruct(cfgString)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
