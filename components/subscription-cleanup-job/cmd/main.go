package main

import (
	"fmt"
	"github.com/kyma-project/control-plane/components/subscription-cleanup-job/internal/cloudprovider"
	"github.com/kyma-project/control-plane/components/subscription-cleanup-job/internal/job"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
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

	secretsInterface, err := newSecretsInterface(cfg)
	exitOnError(err, "Failed to create secrets client ")

	err = job.NewCleaner(secretsInterface, cloudprovider.NewProviderFactory()).Do()
	exitOnError(err, "Job execution failed")

	log.Info("Cleanup job finished successfully!")
}

func exitOnError(err error, context string) {
	if err != nil {
		wrappedError := errors.Wrap(err, context)
		log.Fatal(wrappedError)
	}
}

func newSecretsInterface(cfg config) (corev1.SecretInterface, error) {
	rawKubeconfig, err := ioutil.ReadFile(cfg.Gardener.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Gardener Kubeconfig from path %s: %s", cfg.Gardener.KubeconfigPath, err.Error())
	}

	gardenerClusterConfig, err := clientcmd.RESTConfigFromKubeConfig(rawKubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gardener cluster config: %s", err.Error())
	}

	k8sCoreClientSet, err := kubernetes.NewForConfig(gardenerClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %s", err.Error())
	}

	gardenerNamespace := fmt.Sprintf("garden-%s", cfg.Gardener.Project)
	secretsInterface := k8sCoreClientSet.CoreV1().Secrets(gardenerNamespace)

	return secretsInterface, nil
}
