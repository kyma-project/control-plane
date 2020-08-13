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

	Gardener       GardenerConfig
	DirectorClient DirectorClientConfig
	Kyma           KymaConfig

	KubernetesVersion string `envconfig:"default=1.17.8"`

	QueryLogging bool `envconfig:"default=false"`
}

type KymaConfig struct {
	Version string `envconfig:"default=1.11.0"`
	// PreUpgradeVersion is used in upgrade test
	PreUpgradeVersion string `envconfig:"default=1.10.0"`
}

type GardenerConfig struct {
	Providers   []string `envconfig:"default=Azure"`
	AzureSecret string   `envconfig:"default=''"`
	GCPSecret   string   `envconfig:"default=''"`
}

type DirectorClientConfig struct {
	URL                        string `envconfig:"default=http://compass-director.compass-system.svc.cluster.local:3000/graphql"`
	Namespace                  string `envconfig:"default=kcp-system"`
	OauthCredentialsSecretName string `envconfig:"default=kcp-provisioner-credentials"`
}

func (c TestConfig) String() string {
	return fmt.Sprintf("InternalProvisionerURL=%s, Tenant=%s, "+
		"GardenerProviders=%v GardenerAzureSecret=%v, GardenerGCPSecret=%v, "+
		"DirectorClientURL=%s, DirectorClientNamespace=%s, DirectorClientOauthCredentialsSecretName=%s, "+
		"KuberentesVersion=%s, QueryLogging=%v",
		c.InternalProvisionerURL, c.Tenant,
		c.Gardener.Providers, c.Gardener.AzureSecret, c.Gardener.GCPSecret,
		c.DirectorClient.URL, c.DirectorClient.Namespace, c.DirectorClient.OauthCredentialsSecretName,
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
