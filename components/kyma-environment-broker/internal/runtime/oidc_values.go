package runtime

import (
	"io/ioutil"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func ReadOIDCDefaultValuesFromYAML(yamlFilePath string) (internal.OIDCConfigDTO, error) {
	var values internal.OIDCConfigDTO
	yamlFile, err := ioutil.ReadFile(yamlFilePath)
	if err != nil {
		return internal.OIDCConfigDTO{}, errors.Wrap(err, "while reading YAML file with OIDC default values")
	}

	err = yaml.Unmarshal(yamlFile, &values)
	if err != nil {
		return internal.OIDCConfigDTO{}, errors.Wrap(err, "while unmarshaling YAML file with OIDC default values")
	}
	return values, nil
}
