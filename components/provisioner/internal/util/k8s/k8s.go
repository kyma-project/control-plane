package k8s

import (
	"github.com/pkg/errors"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func ParseToK8sConfig(kubeconfigRaw []byte) (*restclient.Config, error) {
	kubeconfig, err := clientcmd.NewClientConfigFromBytes(kubeconfigRaw)
	if err != nil {
		return nil, errors.Wrap(err, "error constructing kubeconfig from raw config")
	}

	clientConfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get client kubeconfig from parsed config")
	}

	return clientConfig, nil
}
