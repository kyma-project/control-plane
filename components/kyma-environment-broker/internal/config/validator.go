package config

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

// comma separated list of required fields
const requiredFields = "additional-components"

type ConfigMapKeysValidator struct{}

func NewConfigMapKeysValidator() *ConfigMapKeysValidator {
	return &ConfigMapKeysValidator{}
}

func (v *ConfigMapKeysValidator) Validate(cfgString string) error {
	reqs := strings.Split(requiredFields, ",")
	keys, err := v.getKeysFromConfigString(cfgString)
	if err != nil {
		return err
	}
	sort.Strings(reqs)
	sort.Strings(keys)

	var missingConfigs []string
	keysString := strings.Join(keys, ",")
	for _, req := range reqs {
		if !strings.Contains(keysString, req) {
			missingConfigs = append(missingConfigs, req)
		}
	}

	if len(missingConfigs) > 0 {
		return fmt.Errorf("missing required configuration entires: %s", strings.Join(missingConfigs, ","))
	}
	return nil
}

func (v *ConfigMapKeysValidator) getKeysFromConfigString(cfgString string) ([]string, error) {
	keysAndValues := make(map[string]interface{}, 0)
	if err := yaml.Unmarshal([]byte(cfgString), keysAndValues); err != nil {
		return nil, err
	}

	keys := make([]string, 0)
	for k, _ := range keysAndValues {
		keys = append(keys, k)
	}

	return keys, nil
}
