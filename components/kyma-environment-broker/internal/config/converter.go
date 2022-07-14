package config

import "gopkg.in/yaml.v2"

type ConfigMapConverter struct{}

func NewConfigMapConverter() *ConfigMapConverter {
	return &ConfigMapConverter{}
}

func (c *ConfigMapConverter) ConvertToStruct(cfgString string) (ConfigForPlan, error) {
	var cfg ConfigForPlan
	if err := yaml.Unmarshal([]byte(cfgString), &cfg); err != nil {
		return ConfigForPlan{}, err
	}
	return cfg, nil
}
