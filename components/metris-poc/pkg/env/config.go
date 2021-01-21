package env

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type config struct {
	gardenerKubeconfig string `envconfig:"GARDENER_KUBECONFIG" required:"true"`
}

func GetConfig() config {
	cfg := config{}
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	return cfg
}
