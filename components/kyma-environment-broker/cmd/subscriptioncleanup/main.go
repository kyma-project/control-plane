package main

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/subscriptioncleanup/cloudprovider"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/subscriptioncleanup/job"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type config struct {
	Gardener struct {
		KubeconfigPath string `envconfig:"default=/gardener/kubeconfig"`
		Project        string `envconfig:"default="`
	}
}

func main() {
	log.Info("Starting cleanup job!")
	cfg := config{}
	err := envconfig.InitWithPrefix(&cfg, "APP")
	exitOnError(err, "Failed to load application config")

	clusterConfig, err := newClusterConfig(cfg)
	exitOnError(err, "Failed to create kubernetes cluster client")

	kubernetesInterface, err := newKubernetesInterface(clusterConfig)
	exitOnError(err, "Failed to create kubernetes client")

	gardenerClient, err := dynamic.NewForConfig(clusterConfig)
	exitOnError(err, "Failed to create kubernetes client")

	gardenerNamespace := fmt.Sprintf("garden-%s", cfg.Gardener.Project)
	shootInterface := gardenerClient.Resource(gardener.ShootResource).Namespace(gardenerNamespace)
	secretBindingsInterface := gardenerClient.Resource(gardener.SecretBindingResource).Namespace(gardenerNamespace)

	err = job.NewCleaner(context.Background(), kubernetesInterface, secretBindingsInterface, shootInterface, cloudprovider.NewProviderFactory()).Do()
	exitOnError(err, "Job execution failed")

	log.Info("Cleanup job finished successfully!")
}

func exitOnError(err error, context string) {
	if err != nil {
		wrappedError := errors.Wrap(err, context)
		log.Fatal(wrappedError)
	}
}

func newClusterConfig(cfg config) (*restclient.Config, error) {
	rawKubeconfig, err := ioutil.ReadFile(cfg.Gardener.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Gardener Kubeconfig from path %s: %s",
			cfg.Gardener.KubeconfigPath, err.Error())
	}

	gardenerClusterConfig, err := clientcmd.RESTConfigFromKubeConfig(rawKubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gardener cluster config: %s", err.Error())
	}
	return gardenerClusterConfig, nil
}

func newKubernetesInterface(kubeconfig *restclient.Config) (kubernetes.Interface, error) {
	k8sCoreClientSet, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %s", err.Error())
	}
	return k8sCoreClientSet, nil
}
