package gardener

import (
	"fmt"
	"io/ioutil"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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
