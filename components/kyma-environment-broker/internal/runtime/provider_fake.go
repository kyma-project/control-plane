package runtime

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type HTTPFakeClient struct {
	t                 *testing.T
	installerContent  string
	componentsContent string
	code              int

	RequestURL string
}

func (f *HTTPFakeClient) Do(req *http.Request) (*http.Response, error) {
	f.RequestURL = req.URL.String()

	var body io.ReadCloser
	if strings.Contains(f.RequestURL, "kyma-components.yaml") {
		body = ioutil.NopCloser(bytes.NewReader([]byte(f.componentsContent)))
	} else {
		body = ioutil.NopCloser(bytes.NewReader([]byte(f.installerContent)))
	}

	return &http.Response{
		StatusCode: f.code,
		Body:       body,
		Request:    req,
	}, nil
}

func NewTestClient(t *testing.T, installerContent, componentsContent string, code int) *HTTPFakeClient {
	return &HTTPFakeClient{
		t:                 t,
		code:              code,
		installerContent:  installerContent,
		componentsContent: componentsContent,
	}
}

// WithHTTPClient is a helper method to use ONLY in tests
func (r *ComponentsListProvider) WithHTTPClient(doer HTTPDoer) *ComponentsListProvider {
	r.httpClient = doer

	return r
}

// ReadYAMLFromFile is a helper method to use ONLY in tests
func ReadYAMLFromFile(t *testing.T, yamlFileName string) string {
	t.Helper()

	filename := path.Join("testdata", yamlFileName)
	yamlFile, err := ioutil.ReadFile(filename)
	require.NoError(t, err)

	return string(yamlFile)
}
