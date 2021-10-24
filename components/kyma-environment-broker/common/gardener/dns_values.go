package gardener

import (
	"io/ioutil"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func SetGardenerDnsConfig(config *internal.DNSProvidersData, yamlFilePath string) error {
	dnsValues, err := ReadDNSProvidersValuesFromYAML(yamlFilePath)
	if err != nil {
		return err
	}

	config = &dnsValues

	return nil
}

func ReadDNSProvidersValuesFromYAML(yamlFilePath string) (internal.DNSProvidersData, error) {
	var values internal.DNSProvidersData
	yamlFile, err := ioutil.ReadFile(yamlFilePath)
	if err != nil {
		return internal.DNSProvidersData{}, errors.Wrap(err, "while reading YAML file with DNS default values")
	}

	err = yaml.Unmarshal(yamlFile, &values)
	if err != nil {
		return internal.DNSProvidersData{}, errors.Wrap(err, "while unmarshalling YAML file with DNS default values")
	}
	return values, nil
}
