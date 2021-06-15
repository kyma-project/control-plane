package components

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_splitRevision(t *testing.T) {
	for _, s := range []struct {
		URL              string
		expectedPath     string
		expectedRevision string
	}{
		{
			URL:              "github.com/kyma-project/kyma.git?ref=1234abcdx",
			expectedPath:     "github.com/kyma-project/kyma.git",
			expectedRevision: "1234abcdx",
		},
		{
			URL:              "resources/extra-component?ref=1234abcdx",
			expectedPath:     "resources/extra-component",
			expectedRevision: "1234abcdx",
		},
		{
			URL:              "github.com/kyma-project/kyma.git?user=admin",
			expectedPath:     "github.com/kyma-project/kyma.git?user=admin",
			expectedRevision: "",
		},
	} {
		t.Run(s.URL, func(t *testing.T) {
			path, revision, err := splitRevision(s.URL)
			require.NoError(t, err)
			require.Equal(t, s.expectedPath, path)
			require.Equal(t, s.expectedRevision, revision)
		})
	}
}

func Test_splitURL(t *testing.T) {
	for _, s := range []struct {
		URL               string
		expectedURL       string
		expectedComponent string
		expectedRevision  string
	}{
		{
			URL:               "github.com/kyma-project/kyma.git//resources/extra-component?ref=1234abcdx",
			expectedURL:       "https://github.com/kyma-project/kyma.git",
			expectedComponent: "resources/extra-component",
			expectedRevision:  "1234abcdx",
		},
		{
			URL:               "https://github.com/kyma-project/kyma.git//resources/extra-component?ref=1234abcdx",
			expectedURL:       "https://github.com/kyma-project/kyma.git",
			expectedComponent: "resources/extra-component",
			expectedRevision:  "1234abcdx",
		},
		{
			URL:               "http://github.com/kyma-project/kyma.git//resources/extra-component?ref=1234abcdx",
			expectedURL:       "http://github.com/kyma-project/kyma.git",
			expectedComponent: "resources/extra-component",
			expectedRevision:  "1234abcdx",
		},
		{
			URL:               "github.com/kyma-project/kyma.git",
			expectedURL:       "https://github.com/kyma-project/kyma.git",
			expectedComponent: "",
			expectedRevision:  "",
		},
		{
			URL:               "https://github.com/kyma-project/kyma.git//resources/extra-component",
			expectedURL:       "https://github.com/kyma-project/kyma.git",
			expectedComponent: "resources/extra-component",
			expectedRevision:  "",
		},
		{
			URL:               "http://github.com/kyma-project/kyma.git?ref=1234abcdx",
			expectedURL:       "http://github.com/kyma-project/kyma.git",
			expectedComponent: "",
			expectedRevision:  "1234abcdx",
		},
	} {
		t.Run(s.URL, func(t *testing.T) {
			gc := NewGitComponent(s.URL, "", "")
			err := gc.splitURL()
			require.NoError(t, err)
			require.Equal(t, s.expectedURL, gc.src)
			require.Equal(t, s.expectedComponent, gc.component)
			require.Equal(t, s.expectedRevision, gc.revision)
		})
	}
}
