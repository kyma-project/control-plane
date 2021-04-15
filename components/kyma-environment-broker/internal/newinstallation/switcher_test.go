package newinstallation

import (
	"testing"

	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	configMapName      = "new-installation-versions"
	configMapNamespace = "kcp-system"
)

func TestSwitcher_IsNewComponentList(t *testing.T) {
	t.Run("config map with version exist", func(t *testing.T) {
		for name, tc := range map[string]struct {
			version   string
			isNewList bool
		}{
			"new list for version with hash": {
				version:   "main-12345",
				isNewList: true,
			},
			"new list for version with PR": {
				version:   "PR-42",
				isNewList: true,
			},
			"new list for version": {
				version:   "1.23",
				isNewList: true,
			},
			"old list for version": {
				version:   "main-54321",
				isNewList: false,
			},
		} {
			t.Run(name, func(t *testing.T) {
				// given
				sch := runtime.NewScheme()
				require.NoError(t, coreV1.AddToScheme(sch))
				client := fake.NewFakeClientWithScheme(sch, fixConfigMap())

				sw := NewSwitcher(Config{
					ConfigMapName:      configMapName,
					ConfigMapNamespace: configMapNamespace,
				}, client)

				// when
				newList, err := sw.IsNewComponentList(tc.version)
				require.NoError(t, err)

				// then
				require.Equal(t, tc.isNewList, newList)
			})
		}
	})

	t.Run("config map with version not exist", func(t *testing.T) {
		// given
		sch := runtime.NewScheme()
		require.NoError(t, coreV1.AddToScheme(sch))
		client := fake.NewFakeClientWithScheme(sch, &coreV1.ConfigMap{})

		sw := NewSwitcher(Config{
			ConfigMapName:      configMapName,
			ConfigMapNamespace: configMapNamespace,
		}, client)

		// when
		_, err := sw.IsNewComponentList("main-12345")
		require.Error(t, err)
		require.Contains(t, err.Error(), `configmaps "new-installation-versions" not found`)
	})
}

func fixConfigMap() *coreV1.ConfigMap {
	return &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      configMapName,
			Namespace: configMapNamespace,
		},
		Data: map[string]string{
			"main-12345": "true",
			"PR-42":      "true",
			"1.23":       "true",
		},
	}
}
