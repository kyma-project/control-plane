package testkit

import (
	"fmt"
	"log"

	"github.com/pkg/errors"
	"github.com/vrischmann/envconfig"
)

type TestConfig struct {
	InternalProvisionerURL string `envconfig:"default=http://localhost:3000/graphql"`
	Tenant                 string `envconfig:"default=3e64ebae-38b5-46a0-b1ed-9ccee153a0ae"`

	Gardener GardenerConfig
	Kyma     KymaConfig

	KubernetesVersion string `envconfig:"default=1.17.8"`

	QueryLogging bool `envconfig:"default=false"`
}

type KymaConfig struct {
	Version string `envconfig:"default=1.18.0"`
}

type GardenerConfig struct {
	Providers   []string `envconfig:"default=Azure"`
	AzureSecret string   `envconfig:"default=''"`
	GCPSecret   string   `envconfig:"default=''"`
}

func (c TestConfig) String() string {
	return fmt.Sprintf("InternalProvisionerURL=%s, Tenant=%s, "+
		"GardenerProviders=%v GardenerAzureSecret=%v,"+
		"KuberentesVersion=%s, QueryLogging=%v",
		c.InternalProvisionerURL, c.Tenant,
		c.Gardener.Providers, c.Gardener.AzureSecret,
		c.KubernetesVersion, c.QueryLogging)
}

func ReadConfig() (TestConfig, error) {
	cfg := TestConfig{}

	err := envconfig.InitWithPrefix(&cfg, "APP")
	if err != nil {
		return TestConfig{}, errors.Wrap(err, "Error while loading app config")
	}

	log.Printf("Read configuration: %s", cfg.String())
	return cfg, nil
}
