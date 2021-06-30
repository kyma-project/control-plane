package release

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	onDemandVersion = "main-1bcdef"
	kymaVersion     = "1.13.0"
)

func TestOnDemand_DownloadRelease(t *testing.T) {

	t.Run("should download release", func(t *testing.T) {
		for _, testCase := range []struct {
			description string
			version     string
			httpClient  *http.Client
			release     model.Release
		}{
			{
				description: "on demand with Tiller",
				version:     onDemandVersion,
				httpClient: newTestClient(func(req *http.Request) *http.Response {
					assert.Contains(t, req.URL.String(), onDemandVersion)

					if strings.HasSuffix(req.URL.String(), "kyma-installer-cluster.yaml") {
						return installerResponse()
					}
					if strings.HasSuffix(req.URL.String(), "tiller.yaml") {
						return tillerResponse()
					}
					return &http.Response{
						StatusCode: http.StatusBadRequest,
					}
				}),
				release: model.Release{
					Version:       onDemandVersion,
					TillerYAML:    "tiller",
					InstallerYAML: "installer",
				},
			},
			{
				description: "on demand without Tiller",
				version:     onDemandVersion,
				httpClient: newTestClient(func(req *http.Request) *http.Response {
					assert.Contains(t, req.URL.String(), onDemandVersion)

					if strings.HasSuffix(req.URL.String(), "kyma-installer-cluster.yaml") {
						return installerResponse()
					}
					if strings.HasSuffix(req.URL.String(), "tiller.yaml") {
						return notFoundResponse()
					}
					return &http.Response{
						StatusCode: http.StatusBadRequest,
					}
				}),
				release: model.Release{
					Version:       onDemandVersion,
					InstallerYAML: "installer",
				},
			},
			{
				description: "release without Tiller",
				version:     kymaVersion,
				httpClient: newTestClient(func(req *http.Request) *http.Response {
					assert.Contains(t, req.URL.String(), kymaVersion)

					if strings.HasSuffix(req.URL.String(), "kyma-installer-cluster.yaml") {
						return installerResponse()
					}
					if strings.HasSuffix(req.URL.String(), "tiller.yaml") {
						return notFoundResponse()
					}
					return &http.Response{
						StatusCode: http.StatusBadRequest,
					}
				}),
				release: model.Release{
					Version:       kymaVersion,
					InstallerYAML: "installer",
				},
			},
			{
				description: "release with Tiller",
				version:     kymaVersion,
				httpClient: newTestClient(func(req *http.Request) *http.Response {
					assert.Contains(t, req.URL.String(), kymaVersion)

					if strings.HasSuffix(req.URL.String(), "kyma-installer-cluster.yaml") {
						return installerResponse()
					}
					if strings.HasSuffix(req.URL.String(), "tiller.yaml") {
						return tillerResponse()
					}
					return &http.Response{
						StatusCode: http.StatusBadRequest,
					}
				}),
				release: model.Release{
					Version:       kymaVersion,
					InstallerYAML: "installer",
					TillerYAML:    "tiller",
				},
			},
		} {
			t.Run(testCase.description, func(t *testing.T) {
				// given
				fileDownloader := NewFileDownloader(testCase.httpClient)

				onDemand := NewGCSDownloader(fileDownloader)

				// when
				downloadedRelease, err := onDemand.DownloadRelease(testCase.version)
				require.NoError(t, err)

				// then
				assert.Equal(t, testCase.release, downloadedRelease)
			})
		}
	})
}

func tillerResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBufferString("tiller")),
	}
}

func installerResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBufferString("installer")),
	}
}
func notFoundResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       ioutil.NopCloser(bytes.NewBufferString("404 not found")),
	}
}

func TestOnDemand_GetReleaseByVersion_Error(t *testing.T) {

	for _, testCase := range []struct {
		description string
		httpClient  *http.Client
	}{
		{
			description: "should return error when failed to download tiller",
			httpClient: newTestClient(func(req *http.Request) *http.Response {
				if strings.HasSuffix(req.URL.String(), "kyma-installer-cluster.yaml") {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewBufferString("installer")),
					}
				}
				if strings.HasSuffix(req.URL.String(), "tiller.yaml") {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
						Body:       ioutil.NopCloser(bytes.NewBufferString("")),
					}
				}
				return &http.Response{
					StatusCode: http.StatusBadRequest,
				}
			}),
		},
		{
			description: "should return error when failed to download installer",
			httpClient: newTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       ioutil.NopCloser(bytes.NewBufferString("")),
				}
			}),
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			fileDownloader := NewFileDownloader(testCase.httpClient)

			onDemand := NewGCSDownloader(fileDownloader)

			// when
			_, err := onDemand.DownloadRelease(onDemandVersion)

			// then
			require.Error(t, err)
		})
	}

}

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func newTestClient(rtFunc RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(rtFunc),
	}
}
