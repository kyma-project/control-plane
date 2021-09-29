package runtime

import (
	"io/ioutil"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func ReadDNSConfigFromYAML(yamlFilePath string) (internal.DNSConfigDTO, error) {
	var values internal.DNSConfigDTO
	yamlFile, err := ioutil.ReadFile(yamlFilePath)
	if err != nil {
		return internal.DNSConfigDTO{}, errors.Wrap(err, "while reading YAML file with DNS config")
	}

	err = yaml.Unmarshal(yamlFile, &values)
	if err != nil {
		return internal.DNSConfigDTO{}, errors.Wrap(err, "while unmarshalling YAML file with DNS config")
	}
	return values, nil
}
