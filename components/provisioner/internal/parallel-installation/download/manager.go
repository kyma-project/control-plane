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

// Download loads all Kyma resources, installation resources base on revision
// and all external components based on their source URL
// it returns path to resources and installation resources
func (m *Manager) Download(kymaRevision string, kymaComponents []model.KymaComponentConfig) (string, string, error) {
	// lock for goroutines to not download the same packages at the same time
	m.mutex.Lock()

	// download Kyma resources and component from external sources
	resourcesPath, installationResourcesPath, err := m.kymaDownloader.DownloadKyma(m.prepareRevision(kymaRevision))
	if err != nil {
		m.mutex.Unlock()
		return "", "", errors.Wrap(err, "while downloading Kyma resources")
	}

	componentsPath, err := m.componentsDownloader.DownloadExternalComponents(kymaComponents)
	if err != nil {
		m.mutex.Unlock()
		return "", "", errors.Wrap(err, "while downloading components")
	}

	m.mutex.Unlock()

	// copy all downloaded resources to the destination paths
	p := fmt.Sprintf(m.kymaPathTmp, uuid.New().String())
	dstResourcesPath := fmt.Sprintf("%s/resources", p)
	dstInstallPath := fmt.Sprintf("%s/installation-resources", p)

	err = copy.Copy(resourcesPath, dstResourcesPath)
	if err != nil {
		return "", "", errors.Wrap(err, "while copying Kyma resources")
	}
	err = copy.Copy(installationResourcesPath, dstInstallPath)
	if err != nil {
		return "", "", errors.Wrap(err, "while copying Kyma installation resources")
	}
	err = m.copyComponentsToResources(dstResourcesPath, componentsPath, kymaComponents)
	if err != nil {
		return "", "", errors.Wrap(err, "while copying Kyma components")
	}

	return dstResourcesPath, dstInstallPath, nil
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
