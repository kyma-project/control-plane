package provider

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func ReadPlatformRegionMappingFromFile(filename string) (map[string]string, error) {
	regionConfig, err := ioutil.ReadFile(filename)
	if err != nil {
		return map[string]string{}, errors.Wrapf(err, "while reading %s file with region mapping config", filename)
	}
	var data map[string]string
	err = yaml.Unmarshal(regionConfig, &data)
	if err != nil {
		return map[string]string{}, errors.Wrapf(err, "while unmarshalling a file with region mapping config")
	}
	return data, nil
}
