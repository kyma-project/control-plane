package release

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"
)

//go:generate mockery -name=Provider
type Provider interface {
	GetReleaseByVersion(version string) (model.Release, error)
}

//go:generate mockery -name=ReleaseDownloader
type ReleaseDownloader interface {
	DownloadRelease(version string) (model.Release, error)
}

func NewReleaseProvider(repository Repository, downloader ReleaseDownloader) *ReleaseProvider {
	return &ReleaseProvider{
		repository: repository,
		downloader: downloader,
	}
}

type ReleaseProvider struct {
	repository Repository
	downloader ReleaseDownloader
}

func (rp *ReleaseProvider) GetReleaseByVersion(version string) (model.Release, error) {
	release, err := rp.repository.GetReleaseByVersion(version)

	if err == nil { // release found in DB
		return release, nil
	}

	if err.Code() == dberrors.CodeNotFound { // release not found, if is on-demand version, try to download
		return rp.downloadRelease(version)
	}

	return model.Release{}, dberrors.Internal("failed to get Kyma release for version %s: %s", version, err.Error())
}

func (rp *ReleaseProvider) downloadRelease(version string) (model.Release, error) {
	release, err := rp.downloader.DownloadRelease(version)
	if err != nil {
		return model.Release{}, err
	}

	release, err = rp.repository.SaveRelease(release)
	if err != nil {
		return model.Release{}, fmt.Errorf("failed to save Kyma release artifacts for version %s: %s", version, err.Error())
	}
	return release, nil
}
