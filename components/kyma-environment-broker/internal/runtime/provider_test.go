package runtime_test

import (
	"io/ioutil"
	"net/http"
	"path"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestRuntimeComponentProviderGetSuccess(t *testing.T) {
	type given struct {
		kymaVersion                            internal.RuntimeVersionData
		managedRuntimeComponentsYAMLPath       string
		newAdditionalRuntimeComponentsYAMLPath string
	}
	tests := []struct {
		name               string
		given              given
		expectedRequestURL string
	}{
		{
			name: "Provide release Kyma version 1.x",
			given: given{
				kymaVersion:                            internal.RuntimeVersionData{Version: "1.9.0", MajorVersion: 1},
				managedRuntimeComponentsYAMLPath:       path.Join("testdata", "managed-runtime-components.yaml"),
				newAdditionalRuntimeComponentsYAMLPath: path.Join("testdata", "additional-runtime-components.yaml"),
			},
			expectedRequestURL: "https://storage.googleapis.com/kyma-prow-artifacts/1.9.0/kyma-installer-cluster.yaml",
		},
		{
			name: "Provide on-demand Kyma version based on 1.x",
			given: given{
				kymaVersion:                            internal.RuntimeVersionData{Version: "main-ece6e5d9", MajorVersion: 1},
				managedRuntimeComponentsYAMLPath:       path.Join("testdata", "managed-runtime-components.yaml"),
				newAdditionalRuntimeComponentsYAMLPath: path.Join("testdata", "additional-runtime-components.yaml"),
			},
			expectedRequestURL: "https://storage.googleapis.com/kyma-development-artifacts/main-ece6e5d9/kyma-installer-cluster.yaml",
		},
		{
			name: "Provide release Kyma version 2.0.0",
			given: given{
				kymaVersion:                            internal.RuntimeVersionData{Version: "2.0.0", MajorVersion: 2},
				managedRuntimeComponentsYAMLPath:       path.Join("testdata", "managed-runtime-components.yaml"),
				newAdditionalRuntimeComponentsYAMLPath: path.Join("testdata", "additional-runtime-components.yaml"),
			},
			expectedRequestURL: "https://storage.googleapis.com/kyma-prow-artifacts/2.0.0/kyma-components.yaml",
		},
		{
			name: "Provide on-demand Kyma version based on 2.0",
			given: given{
				kymaVersion:                            internal.RuntimeVersionData{Version: "main-ece6e5d9", MajorVersion: 2},
				managedRuntimeComponentsYAMLPath:       path.Join("testdata", "managed-runtime-components.yaml"),
				newAdditionalRuntimeComponentsYAMLPath: path.Join("testdata", "additional-runtime-components.yaml"),
			},
			expectedRequestURL: "https://storage.googleapis.com/kyma-development-artifacts/main-ece6e5d9/kyma-components.yaml",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			installerYAML := runtime.ReadYAMLFromFile(t, "kyma-installer-cluster.yaml")
			componentsYAML := runtime.ReadYAMLFromFile(t, "kyma-components.yaml")
			fakeHTTPClient := runtime.NewTestClient(t, installerYAML, componentsYAML, http.StatusOK)

			listProvider := runtime.NewComponentsListProvider(
				tc.given.managedRuntimeComponentsYAMLPath,
				tc.given.newAdditionalRuntimeComponentsYAMLPath).WithHTTPClient(fakeHTTPClient)

			expAdditionalComponents := make([]internal.KymaComponent, 0)
			if tc.given.kymaVersion.MajorVersion > 1 {
				expAdditionalComponents = readManagedComponentsFromFile(t, tc.given.newAdditionalRuntimeComponentsYAMLPath)
			} else {
				expAdditionalComponents = readManagedComponentsFromFile(t, tc.given.managedRuntimeComponentsYAMLPath)
			}

			// when
			allComponents, err := listProvider.AllComponents(tc.given.kymaVersion, nil)

			// then
			require.NoError(t, err)
			assert.NotNil(t, allComponents)

			assert.Equal(t, tc.expectedRequestURL, fakeHTTPClient.RequestURL)
			assertManagedComponentsAtTheEndOfList(t, allComponents, expAdditionalComponents)
		})
	}
}

func TestRuntimeComponentProviderGetFailures(t *testing.T) {
	type given struct {
		kymaVersion                            internal.RuntimeVersionData
		managedRuntimeComponentsYAMLPath       string
		newAdditionalRuntimeComponentsYAMLPath string
		httpErrMessage                         string
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
				kymaVersion:                            internal.RuntimeVersionData{Version: "111.000.111", MajorVersion: 1},
				managedRuntimeComponentsYAMLPath:       path.Join("testdata", "managed-runtime-components.yaml"),
				newAdditionalRuntimeComponentsYAMLPath: path.Join("testdata", "additional-runtime-components.yaml"),
				httpErrMessage:                         "Not Found",
			},
			returnStatusCode: http.StatusNotFound,
			tempError:        false,
			expErrMessage:    "while getting Kyma components: while checking response status code for Kyma components list: got unexpected status code, want 200, got 404, url: https://storage.googleapis.com/kyma-prow-artifacts/111.000.111/kyma-installer-cluster.yaml, body: Not Found",
		},
		{
			name: "Provide on-demand version not found",
			given: given{
				kymaVersion:                            internal.RuntimeVersionData{Version: "main-123123", MajorVersion: 1},
				managedRuntimeComponentsYAMLPath:       path.Join("testdata", "managed-runtime-components.yaml"),
				newAdditionalRuntimeComponentsYAMLPath: path.Join("testdata", "additional-runtime-components.yaml"),
				httpErrMessage:                         "Not Found",
			},
			returnStatusCode: http.StatusNotFound,
			tempError:        false,
			expErrMessage:    "while getting Kyma components: while checking response status code for Kyma components list: got unexpected status code, want 200, got 404, url: https://storage.googleapis.com/kyma-development-artifacts/main-123123/kyma-installer-cluster.yaml, body: Not Found",
		},
		{
			name: "Provide on-demand version not found, temporary server error",
			given: given{
				kymaVersion:                            internal.RuntimeVersionData{Version: "main-123123", MajorVersion: 1},
				managedRuntimeComponentsYAMLPath:       path.Join("testdata", "managed-runtime-components.yaml"),
				newAdditionalRuntimeComponentsYAMLPath: path.Join("testdata", "additional-runtime-components.yaml"),
				httpErrMessage:                         "Internal Server Error",
			},
			returnStatusCode: http.StatusInternalServerError,
			tempError:        true,
			expErrMessage:    "while getting Kyma components: while checking response status code for Kyma components list: got unexpected status code, want 200, got 500, url: https://storage.googleapis.com/kyma-development-artifacts/main-123123/kyma-installer-cluster.yaml, body: Internal Server Error",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			fakeHTTPClient := runtime.NewTestClient(t, tc.given.httpErrMessage, tc.given.httpErrMessage, tc.returnStatusCode)

			listProvider := runtime.NewComponentsListProvider(
				tc.given.managedRuntimeComponentsYAMLPath,
				tc.given.newAdditionalRuntimeComponentsYAMLPath).
				WithHTTPClient(fakeHTTPClient)

			// when
			components, err := listProvider.AllComponents(tc.given.kymaVersion, nil)

			// then
			assert.Nil(t, components)
			assert.EqualError(t, err, tc.expErrMessage)
			assert.Equal(t, tc.tempError, kebError.IsTemporaryError(err))
		})
	}
}

func assertManagedComponentsAtTheEndOfList(t *testing.T, allComponents, managedComponents []internal.KymaComponent) {
	t.Helper()

	assert.NotPanics(t, func() {
		idx := len(allComponents) - len(managedComponents)
		endOfList := allComponents[idx:]

		assert.Equal(t, endOfList, managedComponents)
	})
}

func readManagedComponentsFromFile(t *testing.T, path string) []internal.KymaComponent {
	t.Helper()

	yamlFile, err := ioutil.ReadFile(path)
	require.NoError(t, err)

	var managedList struct {
		Components []internal.KymaComponent `json:"components"`
	}
	err = yaml.Unmarshal(yamlFile, &managedList)
	require.NoError(t, err)

	return managedList.Components
}
