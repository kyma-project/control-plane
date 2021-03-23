package download

import (
	"fmt"
	"os"

	"github.com/kyma-incubator/hydroform/parallel-install/pkg/git"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type resourcesPath struct {
	resources             string
	installationResources string
}

type Kyma struct {
	kymaURL            string
	destinationPathTmp string
	kymaResources      map[string]resourcesPath
}

func NewKyma(kymaURL string, destinationPathTmp string) *Kyma {
	return &Kyma{
		kymaURL:            kymaURL,
		destinationPathTmp: destinationPathTmp,
		kymaResources:      make(map[string]resourcesPath, 0),
	}
}

func (k *Kyma) DownloadKyma(revision string) (string, string, error) {
	// check if resources for revision exist, return path to resources if yes
	if path, ok := k.kymaResources[revision]; ok {
		return path.resources, path.installationResources, nil
	}

	// cloning Kyma repository to temporary directory
	path := fmt.Sprintf(k.destinationPathTmp, uuid.New().String())
	tmpPath := fmt.Sprintf("%s/tmp", path)
	err := git.CloneRepo(k.kymaURL, tmpPath, revision)
	if err != nil {
		return "", "", errors.Wrapf(err, "while cloning Kyma repository for revision: %s", revision)
	}

	// move resources directory and installation resources directory to the final path
	rp, err := k.moveDirectories(path, tmpPath)
	if err != nil {
		return "", "", errors.Wrap(err, "while moving directories to the final path")
	}

	// remove unnecessary files
	err = os.RemoveAll(tmpPath)
	if err != nil {
		return "", "", errors.Wrapf(err, "while removing tmp directory: %s", tmpPath)
	}

	k.kymaResources[revision] = rp
	return rp.resources, rp.installationResources, nil
}

func (k *Kyma) moveDirectories(path, tmpPath string) (resourcesPath, error) {
	var rp resourcesPath

	rp.resources = fmt.Sprintf("%s/resources", path)
	err := os.Rename(fmt.Sprintf("%s/resources", tmpPath), rp.resources)
	if err != nil {
		return rp, errors.Wrap(err, "while moving resources directory")
	}

	rp.installationResources = fmt.Sprintf("%s/installation-resources", path)
	err = os.Rename(fmt.Sprintf("%s/installation/resources", tmpPath), rp.installationResources)
	if err != nil {
		return rp, errors.Wrap(err, "while moving installation resources directory")
	}

	return rp, nil
}
