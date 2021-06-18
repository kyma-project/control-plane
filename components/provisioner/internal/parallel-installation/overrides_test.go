package parallel_installation

import (
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/parallel-installation/mocks"

	"github.com/stretchr/testify/require"
)

func TestSetOverrides(t *testing.T) {
	t.Run("overrides are properly pass to OverrideBuilder", func(t *testing.T) {
		// given
		overrideBuilder := &mocks.OverrideBuilder{}
		components := []model.KymaComponentConfig{
			{
				Component: "component#1",
				Configuration: model.Configuration{
					ConfigEntries: []model.ConfigEntry{
						model.NewConfigEntry("test1.config1.key1a", "value1a", false),
						model.NewConfigEntry("test1.config1.key1b", "1", false),
					},
				},
			},
			{
				Component: "component#2",
				Configuration: model.Configuration{
					ConfigEntries: []model.ConfigEntry{
						model.NewConfigEntry("test2a.config2a.key2a", "value2a", false),
						model.NewConfigEntry("test2b.config2b.key2b", "value2b", false),
					},
				},
			},
			{
				Component: "component#3",
				Configuration: model.Configuration{
					ConfigEntries: []model.ConfigEntry{
						model.NewConfigEntry("test3.config3.key3", "true", false),
					},
				},
			},
		}
		globalConfiguration := model.Configuration{
			ConfigEntries: []model.ConfigEntry{
				model.NewConfigEntry("global.config.key", "globalValue", false),
			},
		}

		expected := map[string]interface{}{
			"component#1a": map[string]interface{}{
				"test1": map[string]interface{}{
					"config1": map[string]interface{}{
						"key1a": "value1a",
					},
				},
			},
			"component#1b": map[string]interface{}{
				"test1": map[string]interface{}{
					"config1": map[string]interface{}{
						"key1b": "1",
					},
				},
			},
			"component#2a": map[string]interface{}{
				"test2a": map[string]interface{}{
					"config2a": map[string]interface{}{
						"key2a": "value2a",
					},
				},
			},
			"component#2b": map[string]interface{}{
				"test2b": map[string]interface{}{
					"config2b": map[string]interface{}{
						"key2b": "value2b",
					},
				},
			},
			"component#3": map[string]interface{}{
				"test3": map[string]interface{}{
					"config3": map[string]interface{}{
						"key3": "true",
					},
				},
			},
			"global": map[string]interface{}{
				"config": map[string]interface{}{
					"key": "globalValue",
				},
			},
		}

		overrideBuilder.On("AddOverrides", "component#1", expected["component#1a"]).Return(nil)
		overrideBuilder.On("AddOverrides", "component#1", expected["component#1b"]).Return(nil)
		overrideBuilder.On("AddOverrides", "component#2", expected["component#2a"]).Return(nil)
		overrideBuilder.On("AddOverrides", "component#2", expected["component#2b"]).Return(nil)
		overrideBuilder.On("AddOverrides", "component#3", expected["component#3"]).Return(nil)
		overrideBuilder.On("AddOverrides", "global", expected["global"]).Return(nil)

		// when
		err := SetOverrides(overrideBuilder, components, globalConfiguration)

		// then
		require.NoError(t, err)
		overrideBuilder.AssertExpectations(t)
	})

	t.Run("error should occur due to wrong key", func(t *testing.T) {
		// given
		overrideBuilder := &mocks.OverrideBuilder{}
		globalComponents := model.Configuration{
			ConfigEntries: []model.ConfigEntry{
				model.NewConfigEntry(".global.config.key", "globalValue", false),
			},
		}

		// when
		err := SetOverrides(overrideBuilder, []model.KymaComponentConfig{}, globalComponents)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "no key segment can be empty")
	})

	t.Run("error should occur due to empty value", func(t *testing.T) {
		// given
		overrideBuilder := &mocks.OverrideBuilder{}
		components := []model.KymaComponentConfig{
			{
				Component: "component#1",
				Configuration: model.Configuration{
					ConfigEntries: []model.ConfigEntry{
						model.NewConfigEntry("test1.config1.key1", "", false),
					},
				},
			},
		}

		// when
		err := SetOverrides(overrideBuilder, components, model.Configuration{})

		// then
		require.Error(t, err)
		require.Equal(t, err.Error(), `value for key "component#1.test1.config1.key1" not exist/is empty`)
	})
}
