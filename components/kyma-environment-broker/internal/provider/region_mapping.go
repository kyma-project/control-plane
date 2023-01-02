package provider

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

func ReadPlatformRegionMappingFromFile(filename string) (map[string]string, error) {
	regionConfig, err := ioutil.ReadFile(filename)
	if err != nil {
		return map[string]string{}, fmt.Errorf("while reading %s file with region mapping config: %w", filename, err)
	}
	var data map[string]string
	err = yaml.Unmarshal(regionConfig, &data)
	if err != nil {
		return map[string]string{}, fmt.Errorf("while unmarshalling a file with region mapping config: %w", err)
	}
	return data, nil
}
