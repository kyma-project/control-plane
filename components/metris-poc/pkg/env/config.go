package env

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	GardenerKubeconfig string `envconfig:"GARDENER_KUBECONFIG" default:"/gardener/kubeconfig"`
}

func GetConfig() *Config {
	cfg := new(Config)
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	return cfg
}
