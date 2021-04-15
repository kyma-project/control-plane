package newinstallation

import (
	"context"
	"time"

	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Config struct {
	ConfigMapName      string
	ConfigMapNamespace string
}

type Switcher struct {
	config    Config
	k8sClient client.Client
}

func NewSwitcher(cfg Config, k8sClient client.Client) *Switcher {
	return &Switcher{
		config:    cfg,
		k8sClient: k8sClient,
	}
}

// IsNewComponentList checks if the version passed by the parameter is in the list of versions
// which should be install by new installer
func (s Switcher) IsNewComponentList(version string) (bool, error) {
	versionsList, err := s.fetchVersionsData()
	if err != nil {
		return false, errors.Wrap(err, "while fetching versions list")
	}

	for _, v := range versionsList {
		if version == v {
			return true, nil
		}
	}
	return false, nil
}

func (s Switcher) fetchVersionsData() ([]string, error) {
	configMap := &coreV1.ConfigMap{}
	name := client.ObjectKey{Name: s.config.ConfigMapName, Namespace: s.config.ConfigMapNamespace}
	data := make([]string, 0)

	err := wait.PollImmediate(500*time.Millisecond, 5*time.Second, func() (done bool, err error) {
		cmErr := s.k8sClient.Get(context.Background(), name, configMap)
		switch {
		case k8sErr.IsNotFound(cmErr):
			return false, cmErr
		case cmErr != nil:
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return data, errors.Wrapf(err, "while getting config map %s", name)
	}

	for key := range configMap.Data {
		data = append(data, key)
	}
	return data, nil
}
