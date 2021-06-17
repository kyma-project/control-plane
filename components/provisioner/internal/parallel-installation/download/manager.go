package download

import (
	"fmt"
	"strings"
	"sync"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"

	"github.com/google/uuid"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
)

type Config struct {
	KymaURL              string
	ResourcesPathTmp     string
	KymaResourcesPathTmp string
	ComponentsPathTmp    string
}

type Manager struct {
	mutex                sync.Locker
	kymaPathTmp          string
	kymaDownloader       *Kyma
	componentsDownloader *Components
}

func NewManager(mutex sync.Locker, cfg Config) *Manager {
	return &Manager{
		mutex:                mutex,
		kymaPathTmp:          cfg.ResourcesPathTmp,
		kymaDownloader:       NewKyma(cfg.KymaURL, cfg.KymaResourcesPathTmp),
		componentsDownloader: NewComponents(cfg.ComponentsPathTmp),
	}
}

// GetResourcePaths fetches paths to all necessary components from cache (or download if not exist)
// and returns path to resources and installation resources
func (m *Manager) GetResourcePaths(kymaRevision string, kymaComponents []model.KymaComponentConfig) (string, string, error) {
	// fetch paths to Kyma resources and component from external sources (or download if not exist)
	path, installPath, componentsPath, err := m.download(kymaRevision, kymaComponents)
	if err != nil {
		return "", "", errors.Wrap(err, "while downloading resources")
	}

	// copy all downloaded resources to the destination paths
	p := fmt.Sprintf(m.kymaPathTmp, uuid.New().String())
	dstResourcesPath := fmt.Sprintf("%s/resources", p)
	dstInstallPath := fmt.Sprintf("%s/installation-resources", p)

	err = copy.Copy(path, dstResourcesPath)
	if err != nil {
		return "", "", errors.Wrap(err, "while copying Kyma resources")
	}
	err = copy.Copy(installPath, dstInstallPath)
	if err != nil {
		return "", "", errors.Wrap(err, "while copying Kyma installation resources")
	}
	err = m.copyComponentsToResources(dstResourcesPath, componentsPath, kymaComponents)
	if err != nil {
		return "", "", errors.Wrap(err, "while copying Kyma components")
	}

	return dstResourcesPath, dstInstallPath, nil
}

// Download loads all Kyma resources, installation resources base on revision
// and all external components based on their source URL
func (m *Manager) Download(kymaRevision string, kymaComponents []model.KymaComponentConfig) error {
	_, _, _, err := m.download(kymaRevision, kymaComponents)
	if err != nil {
		return errors.Wrap(err, "while downloading resources")
	}

	return nil
}

func (m *Manager) download(revision string, components []model.KymaComponentConfig) (string, string, map[string]string, error) {
	// lock for goroutines to not download the same packages at the same time
	m.mutex.Lock()

	// download Kyma resources and component from external sources
	path, installPath, err := m.kymaDownloader.DownloadKyma(m.prepareRevision(revision))
	if err != nil {
		m.mutex.Unlock()
		return "", "", map[string]string{}, errors.Wrap(err, "while downloading Kyma resources")
	}

	componentsPath, err := m.componentsDownloader.DownloadExternalComponents(components)
	if err != nil {
		m.mutex.Unlock()
		return "", "", map[string]string{}, errors.Wrap(err, "while downloading components")
	}

	m.mutex.Unlock()
	return path, installPath, componentsPath, nil
}

func (m *Manager) copyComponentsToResources(dst string, cPaths map[string]string, components []model.KymaComponentConfig) error {
	for _, component := range components {
		if component.SourceURL == nil {
			continue
		}
		su := *component.SourceURL
		path, ok := cPaths[su]
		if !ok {
			return errors.Errorf("there is no path to component %s", component.Component)
		}
		// copy component to the kyma resources path, if component exist will be replaced
		err := copy.Copy(path, fmt.Sprintf("%s/%s", dst, component.Component))
		if err != nil {
			return errors.Wrapf(err, "while copying %s from %s", component.Component, path)
		}
	}

	return nil
}

// prepareRevision cuts main- part from main-<hash> version because git clone tools expect only hash
func (m *Manager) prepareRevision(version string) string {
	if strings.HasPrefix(version, "main-") {
		return strings.TrimLeft(version, "main-")
	}
	return version
}
