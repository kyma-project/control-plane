package download

import (
	"fmt"
	"strings"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	cmp "github.com/kyma-project/control-plane/components/provisioner/internal/parallel-installation/download/components"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	componentZip = iota
	componentTar
	componentGit
)

type Components struct {
	cacheList          map[string]string
	destinationPathTmp string
}

func NewComponents(destinationPathTmp string) *Components {
	return &Components{
		cacheList:          make(map[string]string, 0),
		destinationPathTmp: destinationPathTmp,
	}
}

// DownloadExternalComponents downloads components based on component.SourceURL value
// and returns list of components with path to the files: map[sourceURL] = path_to_files
func (c *Components) DownloadExternalComponents(components []model.KymaComponentConfig) (map[string]string, error) {
	list := make(map[string]string, 0)
	for _, component := range components {
		if component.SourceURL == nil {
			continue
		}
		sourceURL := *component.SourceURL
		if path, ok := c.cacheList[sourceURL]; ok {
			list[sourceURL] = path
			continue
		}
		path, err := c.downloadComponent(sourceURL)
		if err != nil {
			return list, errors.Wrapf(err, "while downloading component: %s", component.Component)
		}

		c.cacheList[sourceURL] = path
		list[sourceURL] = path
	}

	return list, nil
}

func (c *Components) downloadComponent(URL string) (string, error) {
	var err error
	path := fmt.Sprintf(c.destinationPathTmp, uuid.New().String())

	switch c.detectType(URL) {
	case componentZip:
		err = cmp.DownloadZip(URL, path)
	case componentTar:
		err = cmp.DownloadTgz(URL, path)
	case componentGit:
		gc := cmp.NewGitComponent(URL, c.destinationPathTmp, path)
		err = gc.DownloadGitComponent()
	default:
		return "", errors.Errorf("not supported type for URL: %s", URL)
	}

	if err != nil {
		return "", errors.Wrapf(err, "cannot download package from %s", URL)
	}

	return path, nil
}

func (c *Components) detectType(URL string) int {
	parts := strings.Split(URL, ".")
	switch parts[len(parts)-1] {
	case "zip":
		return componentZip
	case "tgz":
		return componentTar
	default:
		// all other URLs are treated as a repository URL
		return componentGit
	}
}
