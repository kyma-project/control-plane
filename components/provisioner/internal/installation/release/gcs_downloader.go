package release

import (
	"fmt"
	"strings"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"
)

const (
	onDemandTillerFileFormat    = "https://storage.googleapis.com/kyma-development-artifacts/%s/tiller.yaml"
	onDemandInstallerFileFormat = "https://storage.googleapis.com/kyma-development-artifacts/%s/kyma-installer-cluster.yaml"

	releaseInstallerFileFormat = "https://storage.googleapis.com/kyma-prow-artifacts/%s/kyma-installer-cluster.yaml"
	releaseTillerFileFormat    = "https://storage.googleapis.com/kyma-prow-artifacts/%s/tiller.yaml"
)

type TextFileDownloader interface {
	Download(url string) (string, error)
	DownloadOrEmpty(url string) (string, error)
}

// GCSDownloader wraps release.Repository with minimal functionality necessary for downloading the Kyma release from on-demand versions
type GCSDownloader struct {
	downloader TextFileDownloader
}

// NewGCSDownloader returns new instance of GCSDownloader
func NewGCSDownloader(downloader TextFileDownloader) *GCSDownloader {
	return &GCSDownloader{
		downloader: downloader,
	}
}

func (o *GCSDownloader) DownloadRelease(version string) (model.Release, error) {
	tillerURL := fmt.Sprintf(releaseTillerFileFormat, version)
	installerURL := fmt.Sprintf(releaseInstallerFileFormat, version)

	if o.isOnDemandVersion(version) {
		// Download onDemand
		tillerURL = fmt.Sprintf(onDemandTillerFileFormat, version)
		installerURL = fmt.Sprintf(onDemandInstallerFileFormat, version)
	}

	return o.downloadRelease(version, tillerURL, installerURL)
}

// Detection rules:
//   For pull requests: PR-<number>
//   For changes to the main branch: master-<commit_sha>
//   For the latest changes in the main branch: master
//
// source: https://github.com/kyma-project/test-infra/blob/main/docs/prow/prow-architecture.md#generate-development-artifacts
func (o *GCSDownloader) isOnDemandVersion(version string) bool {
	isOnDemandVersion := strings.HasPrefix(version, "PR-") ||
		strings.HasPrefix(version, "master-") ||
		strings.EqualFold(version, "master")
	return isOnDemandVersion
}

func (o *GCSDownloader) downloadRelease(version string, tillerURL, installerURL string) (model.Release, dberrors.Error) {
	tillerYAML, err := o.downloader.DownloadOrEmpty(tillerURL)
	if err != nil {
		return model.Release{}, dberrors.Internal("Failed to download tiller YAML release for version %s: %s", version, err)
	}

	installerYAML, err := o.downloader.Download(installerURL)
	if err != nil {
		return model.Release{}, dberrors.Internal("Failed to download installer YAML release for version %s: %s", version, err)
	}

	rel := model.Release{
		Version:       version,
		TillerYAML:    tillerYAML,
		InstallerYAML: installerYAML,
	}

	return rel, nil
}
