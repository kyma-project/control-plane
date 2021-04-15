package runtime_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
	"testing"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestRuntimeComponentProviderGetSuccess(t *testing.T) {
	type given struct {
		kymaVersion                      string
		managedRuntimeComponentsYAMLPath string
	}
	tests := []struct {
		name                                    string
		given                                   given
		expectedRequestURL                      string
		expectedAmountOfPrerequisitesComponents int
		expectedAmountOfComponents              int
		newComponentList                        bool
	}{
		{
			name: "Provide release Kyma version - old component list",
			given: given{
				kymaVersion:                      "1.9.0",
				managedRuntimeComponentsYAMLPath: path.Join("testdata", "managed-runtime-components.yaml"),
			},
			expectedRequestURL:                      "https://storage.googleapis.com/kyma-prow-artifacts/1.9.0/kyma-installer-cluster.yaml",
			expectedAmountOfPrerequisitesComponents: 0,
			// 30 components from Installation resource list + 3 from managed list
			expectedAmountOfComponents: 33,
			newComponentList:           false,
		},
		{
			name: "Provide on-demand Kyma version - old component list",
			given: given{
				kymaVersion:                      "main-ece6e5d9",
				managedRuntimeComponentsYAMLPath: path.Join("testdata", "managed-runtime-components.yaml"),
			},
			expectedRequestURL:                      "https://storage.googleapis.com/kyma-development-artifacts/main-ece6e5d9/kyma-installer-cluster.yaml",
			expectedAmountOfPrerequisitesComponents: 0,
			// 30 components from Installation resource list + 3 from managed list
			expectedAmountOfComponents: 33,
			newComponentList:           false,
		},
		{
			name: "Provide release Kyma version - new component list",
			given: given{
				kymaVersion:                      "1.9.0",
				managedRuntimeComponentsYAMLPath: path.Join("testdata", "managed-runtime-components.yaml"),
			},
			expectedRequestURL:                      "https://storage.googleapis.com/kyma-prow-artifacts/1.9.0/kyma-components.yaml",
			expectedAmountOfPrerequisitesComponents: 3,
			// 13 components from Kyma list + 3 from managed list
			expectedAmountOfComponents: 16,
			newComponentList:           true,
		},
		{
			name: "Provide on-demand Kyma version - new component list",
			given: given{
				kymaVersion:                      "main-ece6e5d9",
				managedRuntimeComponentsYAMLPath: path.Join("testdata", "managed-runtime-components.yaml"),
			},
			expectedRequestURL:                      "https://storage.googleapis.com/kyma-development-artifacts/main-ece6e5d9/kyma-components.yaml",
			expectedAmountOfPrerequisitesComponents: 3,
			// 13 components from Kyma list + 3 from managed list
			expectedAmountOfComponents: 16,
			newComponentList:           true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			fakeHTTPClient := newTestClient(t, http.StatusOK, "")

			decider := &fixture.FakeListDecider{NewComponentList: tc.newComponentList}
			listProvider := runtime.NewComponentsListProvider(decider, tc.given.managedRuntimeComponentsYAMLPath).WithHTTPClient(fakeHTTPClient)

			expManagedComponents := readManagedComponentsFromFile(t, tc.given.managedRuntimeComponentsYAMLPath)

			// when
			allComponents, err := listProvider.AllComponents(tc.given.kymaVersion)

			// then
			require.NoError(t, err)
			assert.NotNil(t, allComponents)
			assert.Equal(t, allComponents.DefaultNamespace, "kyma-system")
			assert.Len(t, allComponents.Prerequisites, tc.expectedAmountOfPrerequisitesComponents)
			assert.Len(t, allComponents.Components, tc.expectedAmountOfComponents)

			// one of the component has to have not nil component.Source.URL value
			for _, component := range allComponents.Components {
				if component.Source != nil {
					assert.Equal(t, "http://example.com", component.Source.URL)
				}
			}

			assert.Equal(t, tc.expectedRequestURL, fakeHTTPClient.RequestURL)
			assertManagedComponentsAtTheEndOfList(t, allComponents.Components, expManagedComponents)
		})
	}
}

func TestRuntimeComponentProviderGetFailures(t *testing.T) {
	type given struct {
		kymaVersion                      string
		managedRuntimeComponentsYAMLPath string
		httpErrMessage                   string
	}
	tests := []struct {
		name             string
		given            given
		returnStatusCode int
		tempError        bool
		expErrMessage    string
	}{
		{
			name: "Provide release version not found",
			given: given{
				kymaVersion:                      "111.000.111",
				managedRuntimeComponentsYAMLPath: path.Join("testdata", "managed-runtime-components.yaml"),
				httpErrMessage:                   "Not Found",
			},
			returnStatusCode: http.StatusNotFound,
			tempError:        false,
			expErrMessage:    "while getting open source kyma components: while checking response status code for Kyma components list: got unexpected status code, want 200, got 404, url: https://storage.googleapis.com/kyma-prow-artifacts/111.000.111/kyma-components.yaml, body: Not Found",
		},
		{
			name: "Provide on-demand version not found",
			given: given{
				kymaVersion:                      "main-123123",
				managedRuntimeComponentsYAMLPath: path.Join("testdata", "managed-runtime-components.yaml"),
				httpErrMessage:                   "Not Found",
			},
			returnStatusCode: http.StatusNotFound,
			tempError:        false,
			expErrMessage:    "while getting open source kyma components: while checking response status code for Kyma components list: got unexpected status code, want 200, got 404, url: https://storage.googleapis.com/kyma-development-artifacts/main-123123/kyma-components.yaml, body: Not Found",
		},
		{
			name: "Provide on-demand version not found, temporary server error",
			given: given{
				kymaVersion:                      "main-123123",
				managedRuntimeComponentsYAMLPath: path.Join("testdata", "managed-runtime-components.yaml"),
				httpErrMessage:                   "Internal Server Error",
			},
			returnStatusCode: http.StatusInternalServerError,
			tempError:        true,
			expErrMessage:    "while getting open source kyma components: while checking response status code for Kyma components list: got unexpected status code, want 200, got 500, url: https://storage.googleapis.com/kyma-development-artifacts/main-123123/kyma-components.yaml, body: Internal Server Error",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			fakeHTTPClient := newTestClient(t, tc.returnStatusCode, tc.given.httpErrMessage)

			decider := &fixture.FakeListDecider{NewComponentList: true}
			listProvider := runtime.NewComponentsListProvider(decider, tc.given.managedRuntimeComponentsYAMLPath).
				WithHTTPClient(fakeHTTPClient)

			// when
			components, err := listProvider.AllComponents(tc.given.kymaVersion)

			// then
			require.NotNil(t, components)
			assert.Len(t, components.Components, 0)
			assert.EqualError(t, err, tc.expErrMessage)
			assert.Equal(t, tc.tempError, kebError.IsTemporaryError(err))
		})
	}
}

type HTTPFakeClient struct {
	t            *testing.T
	errorMessage string
	code         int

	RequestURL string
}

func newTestClient(t *testing.T, code int, msg string) *HTTPFakeClient {
	return &HTTPFakeClient{
		t:            t,
		code:         code,
		errorMessage: msg,
	}
}

func (f *HTTPFakeClient) Do(req *http.Request) (*http.Response, error) {
	f.RequestURL = req.URL.String()

	var (
		response []byte
		err      error
	)
	if f.errorMessage != "" {
		response = []byte(f.errorMessage)
	} else {
		elements := strings.Split(f.RequestURL, "/")

		filename := path.Join("testdata", elements[len(elements)-1])
		response, err = ioutil.ReadFile(filename)
		require.NoError(f.t, err)
	}

	return &http.Response{
		StatusCode: f.code,
		Body:       ioutil.NopCloser(bytes.NewReader(response)),
		Request:    req,
	}, nil
}

func assertManagedComponentsAtTheEndOfList(t *testing.T, allComponents, managedComponents []runtime.ComponentDefinition) {
	t.Helper()

	assert.NotPanics(t, func() {
		idx := len(allComponents) - len(managedComponents)
		endOfList := allComponents[idx:]

		assert.Equal(t, endOfList, managedComponents)
	})
}

func readManagedComponentsFromFile(t *testing.T, path string) []runtime.ComponentDefinition {
	t.Helper()

	yamlFile, err := ioutil.ReadFile(path)
	require.NoError(t, err)

	var managedList struct {
		Components []runtime.ComponentDefinition `json:"components"`
	}
	err = yaml.Unmarshal(yamlFile, &managedList)
	require.NoError(t, err)

	return managedList.Components
}
