package gardener

import (
	"fmt"
	"io/ioutil"

	gardener_apis "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewGardenerClusterConfig(kubeconfigPath string) (*restclient.Config, error) {

	rawKubeconfig, err := ioutil.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Gardener Kubeconfig from path %s: %s", kubeconfigPath, err.Error())
	}

	gardenerClusterConfig, err := RESTConfig(rawKubeconfig)
	if err != nil {
		return nil, fmt.Errorf("")
	}

	return gardenerClusterConfig, nil
}

func NewGardenerSecretBindingsInterface(gardenerClient *gardener_apis.CoreV1beta1Client, gardenerProjectName string) gardener_apis.SecretBindingInterface {
	gardenerNamespace := gardenerNamespace(gardenerProjectName)
	return gardenerClient.SecretBindings(gardenerNamespace)
}

func NewGardenerShootInterface(gardenerClusterCfg *restclient.Config, gardenerProjectName string) (gardener_apis.ShootInterface, error) {

	gardenerNamespace := gardenerNamespace(gardenerProjectName)

	gardenerClusterClient, err := gardener_apis.NewForConfig(gardenerClusterCfg)
	if err != nil {
		return nil, err
	}

	return gardenerClusterClient.Shoots(gardenerNamespace), nil
}

func RESTConfig(kubeconfig []byte) (*restclient.Config, error) {
	return clientcmd.RESTConfigFromKubeConfig(kubeconfig)
}

func NewClient(config *restclient.Config) (*gardener_apis.CoreV1beta1Client, error) {
	clientset, err := gardener_apis.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func gardenerNamespace(projectName string) string {
	return fmt.Sprintf("garden-%s", projectName)
}
