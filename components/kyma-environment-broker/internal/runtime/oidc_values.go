package runtime

import (
	"fmt"
	"io/ioutil"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"gopkg.in/yaml.v2"
)

func ReadOIDCDefaultValuesFromYAML(yamlFilePath string) (internal.OIDCConfigDTO, error) {
	var values internal.OIDCConfigDTO
	yamlFile, err := ioutil.ReadFile(yamlFilePath)
	if err != nil {
		return internal.OIDCConfigDTO{}, fmt.Errorf("while reading YAML file with OIDC default values: %w", err)
	}

	err = yaml.Unmarshal(yamlFile, &values)
	if err != nil {
		return internal.OIDCConfigDTO{}, fmt.Errorf("while unmarshalling YAML file with OIDC default values: %w", err)
	}
	return values, nil
}
