package cls

import (
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/gobuffalo/packr"
	"gopkg.in/yaml.v2"
)

//go:generate packr -v


var (
	defaultConfig = Config{
		RetentionPeriod:    7,
		MaxDataInstances:   2,
		MaxIngestInstances: 2,
	}
)

//Config is the top-level CLS provisioning configuration
type Config struct {
	//Log retention period specified in days
	RetentionPeriod int `yaml:"retentionPeriod"`

	//Number of Elasticsearch data nodes to be provisioned
	MaxDataInstances int `yaml:"maxDataInstances"`

	//Number of FluentD instances to be provisioned
	MaxIngestInstances int `yaml:"maxIngestInstances"`

	SAML *SAMLConfig `yaml:"saml"`

	ServiceManager *ServiceManagerConfig `yaml:"serviceManager"`
}

type OverrideFiles struct {
	clsFBOverrideConfig *template.Template
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

// SAMLConfig to be used by Kibana
type SAMLConfig struct {
	//New admin backend role that maps to any of your SAML group. It will have the right to modify the security module
	AdminGroup string `yaml:"admin_group"`

	//Set to true to use IdP-initiated SSO
	Initiated bool `yaml:"initiated"`

	//The key to sign tokens
	ExchangeKey string `yaml:"exchange_key"`

	//The list of backend_roles will be read from this attribute
	RolesKey string `yaml:"roles_key"`

	Idp *SAMLIdpConfig `yaml:"idp"`

	Sp *SAMLSpConfig `yaml:"sp"`
}

//SAMLIdpConfig contains SAML identity provider configuration
type SAMLIdpConfig struct {
	//URL to get the SAML metadata
	MetadataURL string `yaml:"metadata_url"`

	//SAML entity id
	EntityID string `yaml:"entity_id"`
}

//SAMLSpConfig contains SAML service provider configuration
type SAMLSpConfig struct {
	//Entity ID of the service provider
	EntityID string `yaml:"entity_id"`

	//The private key used to sign the requests (base64 encoded)
	SignaturePrivateKey string `yaml:"signature_private_key"`
}

// Load parses the YAML input s into a Config
func Load(s string) (*Config, error) {
	config := defaultConfig

	if err := yaml.UnmarshalStrict([]byte(s), &config); err != nil {
		return nil, err
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	return &config, nil
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

	if c.SAML == nil {
		return errors.New("no saml")
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

func ParseTemplate() (*template.Template, error) {
	box := packr.NewBox("./templates")
	yamlFile, err := box.FindString("cls_override.yaml")
	tpl, err := template.New("cls_override").Parse(yamlFile)
	if err != nil {
		return nil, err
	}
	return tpl, nil

}
