package cls

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

//Config is the top-level CLS provisioning configuration
type Config struct {
	ServiceManager *ServiceManagerConfig `yaml:"serviceManager"`
}

//ServiceManagerConfig contains service manager credentials per region
type ServiceManagerConfig struct {
	Credentials []*ServiceManagerCredentials `yaml:"credentials"`
}

//ServiceManagerCredentials contains basic auth credentials for a ServiceManager tenant in a particular region
type ServiceManagerCredentials struct {
	Region   Region `yaml:"region"`
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

//Region represents an SAP Cloud Platform region, where a CLS instance can be provisioned
type Region string

//Supported regions
const (
	RegionEurope Region = "eu"
	RegionUS     Region = "us"
)

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
	if c.ServiceManager == nil || len(c.ServiceManager.Credentials) == 0 {
		return errors.New("no service manager credentials")
	}

	for _, creds := range c.ServiceManager.Credentials {
		if err := creds.validate(); err != nil {
			return fmt.Errorf("service manager credentials: %v", err)
		}
	}

	return nil
}

func (c *ServiceManagerCredentials) validate() error {
	if len(c.Region) == 0 {
		return errors.New("no region")
	}

	if err := c.Region.validate(); err != nil {
		return err
	}

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

func (r Region) validate() error {
	supportedRegions := []string{string(RegionEurope), string(RegionUS)}
	for _, sr := range supportedRegions {
		if sr == string(r) {
			return nil
		}
	}

	return fmt.Errorf("unsupported region: %s (%s supported only)", r, strings.Join(supportedRegions, ","))
}
