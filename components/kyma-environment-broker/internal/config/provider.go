package config

import "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

type (
	ConfigReader interface {
		Read(kymaVersion, planName string) (string, error)
	}

	ConfigValidator interface {
		Validate(cfgString string) error
	}

	ConfigConverter interface {
		ConvertToStruct(cfgString string) (internal.ConfigForPlan, error)
	}
)

type ConfigProvider struct {
	Reader    ConfigReader
	Validator ConfigValidator
	Converter ConfigConverter
}

func NewConfigProvider(reader ConfigReader, validator ConfigValidator, converter ConfigConverter) *ConfigProvider {
	return &ConfigProvider{Reader: reader, Validator: validator, Converter: converter}
}

func (p *ConfigProvider) ProvideForGivenVersionAndPlan(kymaVersion, planName string) (*internal.ConfigForPlan, error) {
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
