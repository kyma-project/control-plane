package gardener

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func ReadDNSProvidersValuesFromYAML(yamlFilePath string) (DNSProvidersData, error) {
	var values DNSProvidersData
	yamlFile, err := ioutil.ReadFile(yamlFilePath)
	if err != nil {
		return DNSProvidersData{}, errors.Wrap(err, "while reading YAML file with DNS default values")
	}

	err = yaml.Unmarshal(yamlFile, &values)
	if err != nil {
		return DNSProvidersData{}, errors.Wrap(err, "while unmarshalling YAML file with DNS default values")
	}

	return values, nil
}
