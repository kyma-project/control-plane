package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/kyma-project/control-plane/docs/internal/investigations/runtime-governor-poc/runtime/component/internal/dapr"
	"github.com/kyma-project/control-plane/docs/internal/investigations/runtime-governor-poc/runtime/component/internal/kcp"

	"github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/vrischmann/envconfig"
)

const (
	selectorMetadataName = "selector"
)

type Config struct {
	Interval       time.Duration `envconfig:"default=1s,optional"`
	KubeconfigPath string        `envconfig:"optional"`
	RuntimeID      string        `envconfig:"default=1"`
	URL            string        `envconfig:"default=https://runtime-governor.kyma.local"`
}

func main() {
	cfg := Config{}
	err := envconfig.InitWithPrefix(&cfg, "APP")
	if err != nil {
		panic(err)
	}

	kcpCli := kcp.NewClient(cfg.URL)
	daprCli := dapr.NewClientOrDie(cfg.KubeconfigPath)

	for {
		time.Sleep(cfg.Interval)

		resource, err := kcpCli.Fetch(cfg.RuntimeID)
		if err != nil {
			log.Println("Error when fetching the configuration data")
			log.Println(err.Error())
			continue
		}

		reload, err := daprCli.UpsertComponent(resource, resource.Namespace)
		if err != nil {
			log.Println("Error when upserting the dapr Component")
			log.Println(err.Error())
			continue
		}

		if reload {
			selector, err := getSelectorFromMetadata(resource.Spec.Metadata)
			if err != nil {
				log.Println("Error when parsing the selector from resource metadata")
				log.Println(err.Error())
				continue
			}

			if err := daprCli.DeletePodsForSelector(selector, resource.Namespace); err != nil {
				log.Printf("Error when deleting pods for given selector (%s)\n", selector)
				log.Println(err.Error())
				continue
			}

			log.Println("Successfully reloaded configuration")
		}
	}

	fmt.Println("Finished successfully!")
}

func getSelectorFromMetadata(metadata []v1alpha1.MetadataItem) (string, error) {
	for _, item := range metadata {
		if item.Name == selectorMetadataName {
			return item.Value, nil
		}
	}
	return "", errors.New("selector not found")
}
