package release

import (
	"fmt"
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/installation/release/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

func TestReleaseProvider_GetReleaseByVersion(t *testing.T) {

	release := model.Release{
		Id:            "abcd-efgh",
		Version:       kymaVersion,
		TillerYAML:    "tiller",
		InstallerYAML: "installer",
	}

	t.Run("should get release form database", func(t *testing.T) {
		// given
		repo := &mocks.Repository{}
		repo.On("GetReleaseByVersion", kymaVersion).Return(release, nil)
		downloader := &mocks.ReleaseDownloader{}

		relProvider := NewReleaseProvider(repo, downloader)

		// when
		providedRel, err := relProvider.GetReleaseByVersion(kymaVersion)
		require.NoError(t, err)

		// then
		assert.Equal(t, release, providedRel)
		repo.AssertExpectations(t)
	})

	t.Run("should download release if not found in database", func(t *testing.T) {
		// given
		repo := &mocks.Repository{}
		repo.On("GetReleaseByVersion", kymaVersion).Return(model.Release{}, dberrors.NotFound("error"))
		repo.On("SaveRelease", release).Return(release, nil)

		downloader := &mocks.ReleaseDownloader{}
		downloader.On("DownloadRelease", kymaVersion).Return(release, nil)

		relProvider := NewReleaseProvider(repo, downloader)

		// when
		providedRel, err := relProvider.GetReleaseByVersion(kymaVersion)
		require.NoError(t, err)

		// then
		assert.Equal(t, release, providedRel)
		repo.AssertExpectations(t)
		downloader.AssertExpectations(t)
	})

	t.Run("should get artifacts from database on conflict", func(t *testing.T) {
		// given
		repo := &mocks.Repository{}
		repo.On("GetReleaseByVersion", kymaVersion).Return(model.Release{}, dberrors.NotFound("error")).Once()
		repo.On("SaveRelease", release).Return(model.Release{}, dberrors.AlreadyExists("error"))
		repo.On("GetReleaseByVersion", kymaVersion).Return(release, nil).Once()

		downloader := &mocks.ReleaseDownloader{}
		downloader.On("DownloadRelease", kymaVersion).Return(release, nil)

		relProvider := NewReleaseProvider(repo, downloader)

		// when
		providedRel, err := relProvider.GetReleaseByVersion(kymaVersion)
		require.NoError(t, err)

		// then
		assert.Equal(t, release, providedRel)
		repo.AssertExpectations(t)
		downloader.AssertExpectations(t)
	})

}

func TestReleaseProvider_GetReleaseByVersionError(t *testing.T) {

	t.Run("should return error then failed to get release from database", func(t *testing.T) {
		// given
		repo := &mocks.Repository{}
		repo.On("GetReleaseByVersion", kymaVersion).Return(model.Release{}, dberrors.Internal("error"))
		downloader := &mocks.ReleaseDownloader{}

		relProvider := NewReleaseProvider(repo, downloader)

		// when
		_, err := relProvider.GetReleaseByVersion(kymaVersion)
		require.Error(t, err)
	})

	t.Run("should return error then failed to download release", func(t *testing.T) {
		// given
		repo := &mocks.Repository{}
		repo.On("GetReleaseByVersion", kymaVersion).Return(model.Release{}, dberrors.NotFound("error"))
		downloader := &mocks.ReleaseDownloader{}
		downloader.On("DownloadRelease", kymaVersion).Return(model.Release{}, fmt.Errorf("error"))

		relProvider := NewReleaseProvider(repo, downloader)

		// when
		_, err := relProvider.GetReleaseByVersion(kymaVersion)
		require.Error(t, err)
	})

	t.Run("should return error when failed to save the release", func(t *testing.T) {
		// given
		repo := &mocks.Repository{}
		repo.On("GetReleaseByVersion", kymaVersion).Return(model.Release{}, dberrors.NotFound("error"))
		repo.On("SaveRelease", model.Release{}).Return(model.Release{}, dberrors.Internal("error"))
		downloader := &mocks.ReleaseDownloader{}
		downloader.On("DownloadRelease", kymaVersion).Return(model.Release{}, nil)

		relProvider := NewReleaseProvider(repo, downloader)

		// when
		_, err := relProvider.GetReleaseByVersion(kymaVersion)
		require.Error(t, err)
	})

}
