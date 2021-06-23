package runtime

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type OIDCInputProvider struct {
	oidcDefaultValuesYAMLPath string
	values                    map[string]string
}

func NewOIDCInputProvider(oidcDefaultValuesYAMLPath string) *OIDCInputProvider {
	return &OIDCInputProvider{
		oidcDefaultValuesYAMLPath: oidcDefaultValuesYAMLPath,
		values:                    make(map[string]string, 0),
	}
}

func (p *OIDCInputProvider) Defaults() (map[string]string, error) {
	yamlFile, err := ioutil.ReadFile(p.oidcDefaultValuesYAMLPath)
	if err != nil {
		return nil, errors.Wrap(err, "while reading YAML file with OIDC default values")
	}
	err = yaml.Unmarshal(yamlFile, p.values)
	if err != nil {
		return nil, errors.Wrap(err, "while unmarshaling YAML file with OIDC default values")
	}
	return p.values, nil
}
