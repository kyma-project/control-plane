package release

import (
	log "github.com/sirupsen/logrus"

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

	release, dberr := rp.repository.SaveRelease(release)
	if dberr != nil {
		// The Artifacts could have been saved by different thread while this one was downloading them
		// In such case return artifacts from DB
		if dberr.Code() == dberrors.CodeAlreadyExists {
			log.Warnf("Artifacts for %s version already exist: %s. Fetching artifacts from database.", version, dberr.Error())
			return rp.repository.GetReleaseByVersion(version)
		}
		return model.Release{}, dberr
	}
	return release, nil
}
