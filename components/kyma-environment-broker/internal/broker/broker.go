package broker

import (
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
)

const (
	KymaServiceID   = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	KymaServiceName = "kymaruntime"
)

type PlanDefaults func(planID string, platformProvider internal.CloudProvider, parametersProvider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error)

type KymaEnvironmentBroker struct {
	*ServicesEndpoint
	*ProvisionEndpoint
	*DeprovisionEndpoint
	*UpdateEndpoint
	*GetInstanceEndpoint
	*LastOperationEndpoint
	*BindEndpoint
	*UnbindEndpoint
	*GetBindingEndpoint
	*LastBindingOperationEndpoint
}

// Config represents configuration for broker
type Config struct {
	EnablePlans                       EnablePlans `envconfig:"default=azure"`
	OnlySingleTrialPerGA              bool        `envconfig:"default=true"`
	URL                               string
	EnableKubeconfigURLLabel          bool   `envconfig:"default=false"`
	IncludeAdditionalParamsInSchema   bool   `envconfig:"default=false"`
	ShowTrialExpireInfo               bool   `envconfig:"default=false"`
	SubaccountIDToShowTrialExpireInfo string `envconfig:"default="`
	TrialDocsURL                      string `envconfig:"default="`
}

type ServicesConfig map[string]Service

func NewServicesConfigFromFile(path string) (ServicesConfig, error) {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "while reading YAML file with managed components list")
	}
	var servicesConfig struct {
		Services ServicesConfig `yaml:"services"`
	}
	err = yaml.Unmarshal(yamlFile, &servicesConfig)
	if err != nil {
		return nil, errors.Wrap(err, "while unmarshaling YAML file with managed components list")
	}
	return servicesConfig.Services, nil
}

func (s ServicesConfig) DefaultPlansConfig() (PlansConfig, error) {
	cfg, ok := s[KymaServiceName]
	if !ok {
		return nil, errors.Errorf("while getting data about %s plans", KymaServiceName)
	}
	return cfg.Plans, nil
}

type Service struct {
	Description string          `yaml:"description"`
	Metadata    ServiceMetadata `yaml:"metadata"`
	Plans       PlansConfig     `yaml:"plans"`
}

type ServiceMetadata struct {
	DisplayName         string `yaml:"displayName"`
	ImageUrl            string `yaml:"imageUrl"`
	LongDescription     string `yaml:"longDescription"`
	ProviderDisplayName string `yaml:"providerDisplayName"`
	DocumentationUrl    string `yaml:"documentationUrl"`
	SupportUrl          string `yaml:"supportUrl"`
}

type PlansConfig map[string]PlanData

type PlanData struct {
	Description string       `yaml:"description"`
	Metadata    PlanMetadata `yaml:"metadata"`
}
type PlanMetadata struct {
	DisplayName string `yaml:"displayName"`
}

// EnablePlans defines the plans that should be available for provisioning
type EnablePlans []string

// Unmarshal provides custom parsing of enabled plans.
// Implements envconfig.Unmarshal interface.
func (m *EnablePlans) Unmarshal(in string) error {
	plans := strings.Split(in, ",")
	for _, name := range plans {
		if _, exists := PlanIDsMapping[name]; !exists {
			return errors.Errorf("unrecognized %v plan name ", name)
		}
	}

	*m = plans
	return nil
}
