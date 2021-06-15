package download

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/require"
)

func TestComponents_DownloadExternalComponents(t *testing.T) {
	// Given
	testDir := "./components/test"
	err := copy.Copy("./components/testdata", testDir)
	require.NoError(t, err)

	defer func() {
		err = os.RemoveAll(testDir)
		require.NoError(t, err)
	}()

	// When
	cmp := NewComponents(testDir + "/%s")
	paths, err := cmp.DownloadExternalComponents([]model.KymaComponentConfig{
		{
			Component: "test-zip",
			SourceURL: stringPtr(fmt.Sprintf("%s/simple-chart.zip", testDir)),
		},
		{
			Component: "test-tgz",
			SourceURL: stringPtr(fmt.Sprintf("%s/simple-chart.tgz", testDir)),
		},
	})
	require.NoError(t, err)

	// Then
	require.Len(t, paths, 2)
	require.ElementsMatch(t,
		[]string{
			fmt.Sprintf("%s/simple-chart.zip", testDir),
			fmt.Sprintf("%s/simple-chart.tgz", testDir),
		},
		getKeysFromMap(paths))

	requiredFiles := newRequiredFiles()
	for _, path := range paths {
		counter := 0
		err = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
			if requiredFiles.fileExist(d.Name()) {
				counter++
			}
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, counter, requiredFiles.Len())
	}
}

func stringPtr(str string) *string {
	return &str
}

func getKeysFromMap(m map[string]string) []string {
	keys := make([]string, 0)
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

type requiredFiles struct {
	files []string
}

func newRequiredFiles() *requiredFiles {
	return &requiredFiles{files: []string{
		"Chart.yaml",
		"values.yaml",
		"_helpers.tpl",
		"deployment.yaml",
		"config-map.yaml",
		"rbac.yaml",
	}}
}

func (rf *requiredFiles) fileExist(file string) bool {
	for _, f := range rf.files {
		if f == file {
			return true
		}
	}
	return false
}

func (rf *requiredFiles) Len() int {
	return len(rf.files)
}
