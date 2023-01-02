package gardener

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

func ReadDNSProvidersValuesFromYAML(yamlFilePath string) (DNSProvidersData, error) {
	var values DNSProvidersData
	yamlFile, err := ioutil.ReadFile(yamlFilePath)
	if err != nil {
		return DNSProvidersData{}, fmt.Errorf("while reading YAML file with DNS default values: %w", err)
	}

	err = yaml.Unmarshal(yamlFile, &values)
	if err != nil {
		return DNSProvidersData{}, fmt.Errorf("while unmarshalling YAML file with DNS default values: %w", err)

	}

	return values, nil
}
