package gardener

import (
	"fmt"
	"io/ioutil"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	gardener_apis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
)

type SecretBinding struct {
	unstructured.Unstructured
}

func (b SecretBinding) GetSecretRefName() string {
	str, _, err := unstructured.NestedString(b.Unstructured.Object, "secretRef", "name")
	if err != nil {
		// NOTE this is a safety net, gardener v1beta1 API would need to break the contract for this to panic
		panic(fmt.Sprintf("SecretBinding missing field '.secretRef.name': %v", err))
	}
	return str
}

type Shoot struct {
	unstructured.Unstructured
}

func (b Shoot) GetSpecSecretBindingName() string {
	str, _, err := unstructured.NestedString(b.Unstructured.Object, "spec", "secretBindingName")
	if err != nil {
		// NOTE this is a safety net, gardener v1beta1 API would need to break the contract for this to panic
		panic(fmt.Sprintf("Shoot missing field '.spec.secretBindingName': %v", err))
	}
	return str
}

func (b Shoot) GetSpecMaintenanceTimeWindowBegin() string {
	str, _, err := unstructured.NestedString(b.Unstructured.Object, "spec", "maintenance", "timeWindow", "begin")
	if err != nil {
		// NOTE this is a safety net, gardener v1beta1 API would need to break the contract for this to panic
		panic(fmt.Sprintf("Shoot missing field '.spec.maintenance.timeWindow.begin': %v", err))
	}
	return str
}

func (b Shoot) GetSpecMaintenanceTimeWindowEnd() string {
	str, _, err := unstructured.NestedString(b.Unstructured.Object, "spec", "maintenance", "timeWindow", "end")
	if err != nil {
		// NOTE this is a safety net, gardener v1beta1 API would need to break the contract for this to panic
		panic(fmt.Sprintf("Shoot missing field '.spec.maintenance.timeWindow.end': %v", err))
	}
	return str
}

func (b Shoot) GetSpecRegion() string {
	str, _, err := unstructured.NestedString(b.Unstructured.Object, "spec", "region")
	if err != nil {
		// NOTE this is a safety net, gardener v1beta1 API would need to break the contract for this to panic
		panic(fmt.Sprintf("Shoot missing field '.spec.region': %v", err))
	}
	return str
}

var SecretBindingResource = schema.GroupVersionResource{Group: "core.gardener.cloud", Version: "v1beta1", Resource: "secretbindings"}
var ShootResource = schema.GroupVersionResource{Group: "core.gardener.cloud", Version: "v1beta1", Resource: "shoots"}

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

func RESTConfig(kubeconfig []byte) (*restclient.Config, error) {
	return clientcmd.RESTConfigFromKubeConfig(kubeconfig)
}

// NOTE: for subscription cleanup job backwards compatibility

func gardenerNamespace(projectName string) string {
	return fmt.Sprintf("garden-%s", projectName)
}

func NewClient(config *restclient.Config) (*gardener_apis.CoreV1beta1Client, error) {
	clientset, err := gardener_apis.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
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
