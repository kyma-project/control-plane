package installation

import (
	"fmt"
	"strings"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"

	"github.com/pkg/errors"
)

//go:generate mockery -name=OverrideBuilder
type OverrideBuilder interface {
	AddOverrides(string, map[string]interface{}) error
}

func SetOverrides(ob OverrideBuilder, components []model.KymaComponentConfig, globalConfiguration model.Configuration) error {
	for _, override := range convertToOverrides(components, globalConfiguration) {
		pair := strings.SplitN(override, "=", 2)
		if len(pair) != 2 {
			return errors.Errorf("key/value pair does not have the required amount of value")
		}

		if pair[0] == "" {
			return errors.Errorf("key for override %q not exist/is empty", override)
		}
		if pair[1] == "" {
			return errors.Errorf("value for key %q not exist/is empty", pair[0])
		}

		comp, overridesMap, err := convertToOverridesMap(pair[0], pair[1])
		if err != nil {
			return errors.Wrap(err, "while converting key/value to override map")
		}

		if err := ob.AddOverrides(comp, overridesMap); err != nil {
			return errors.Wrapf(err, "while adding override for %s component", comp)
		}

	}
	return nil
}

func convertToOverrides(components []model.KymaComponentConfig, globalConfiguration model.Configuration) []string {
	overrides := make([]string, 0)

	for _, component := range components {
		name := component.Component
		for _, ov := range component.Configuration.ConfigEntries {
			overrides = append(overrides, fmt.Sprintf("%s.%s=%s", name, ov.Key, ov.Value))
		}
	}

	for _, global := range globalConfiguration.ConfigEntries {
		overrides = append(overrides, fmt.Sprintf("%s=%s", global.Key, global.Value))
	}

	return overrides
}

func convertToOverridesMap(key, value string) (string, map[string]interface{}, error) {
	var comp string
	var latestOverrideMap map[string]interface{}

	keyTokens := strings.Split(key, ".")
	if len(keyTokens) < 2 {
		return comp, latestOverrideMap, fmt.Errorf("override key must contain at least the chart name "+
			"and one override: chart.override[.suboverride]=value (given was '%s=%s')", key, value)
	}

	// first token in key is the chart name
	comp = keyTokens[0]

	// use the remaining key-tokens to build the nested overrides map
	// processing starts from last element to the beginning
	for idx := range keyTokens[1:] {
		overrideMap := make(map[string]interface{})     // current override-map
		overrideName := keyTokens[len(keyTokens)-1-idx] // get last token element
		if idx == 0 {
			// this is the last key-token, use it value
			overrideMap[overrideName] = value
		} else {
			// the latest override map has to become a sub-map of the current override-map
			overrideMap[overrideName] = latestOverrideMap
		}
		//set the current override map as latest override map
		latestOverrideMap = overrideMap
	}

	if len(latestOverrideMap) < 1 {
		return comp, latestOverrideMap, fmt.Errorf("failed to extract overrides map from '%s=%s'", key, value)
	}

	return comp, latestOverrideMap, nil
}
