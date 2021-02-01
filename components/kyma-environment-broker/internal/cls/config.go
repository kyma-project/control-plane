package cls

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v2"
)

type CreateInstanceInput struct {
	Name            string
	Region          string
	GlobalAccountID string
}

type CreateInstanceOutput struct {
	ID string `json:"id"`
}

//Config is the top-level CLS provisioning configuration
type Config struct {
	ServiceManagerCredentials ServiceManagerCredentials `yaml:"serviceManagerCredentials"`
}

//ServiceManagerCredentials contains basic auth credentials for ServiceManager in different regions
type ServiceManagerCredentials struct {
	Regions map[string]RegionServiceManagerCredentials `yaml:"regions"`
}

//RegionServiceManagerCredentials contains basic auth credentials for ServiceManager in a particular region
type RegionServiceManagerCredentials struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Load parses the YAML input s into a Config
func Load(s string) (*Config, error) {
	config := &Config{}

	if err := yaml.UnmarshalStrict([]byte(s), config); err != nil {
		return nil, err
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	return config, nil
}

func (c *Config) validate() error {
	if len(c.ServiceManagerCredentials.Regions) == 0 {
		return errors.New("no service manager credentials")
	}

	for _, creds := range c.ServiceManagerCredentials.Regions {
		if err := creds.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c *RegionServiceManagerCredentials) validate() error {
	if len(c.URL) == 0 {
		return errors.New("no url")
	}

	if len(c.Username) == 0 {
		return errors.New("no username")
	}

	if len(c.Password) == 0 {
		return errors.New("no password")
	}

	return nil
}
