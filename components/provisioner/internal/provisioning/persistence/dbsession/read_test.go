package dbsession

import (
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseToKymaConfig(t *testing.T) {

	kymaConfigId := "abc-kyma-def"
	runtimeId := "abc-runtime-def"
	releaseId := "abc-release-def"
	version := "18.0.0"
	tillerYaml := "tiller"
	installerYaml := "installer"
	profileProduction := string(model.ProductionProfile)
	expectedProfileProduction := model.ProductionProfile

	newComponentDTO := func(id, name, namespace string, profile *string, order int) kymaComponentConfigDTO {
		return kymaComponentConfigDTO{
			ID:                  id,
			KymaConfigID:        kymaConfigId,
			ReleaseID:           releaseId,
			Profile:             profile,
			Version:             version,
			TillerYAML:          tillerYaml,
			InstallerYAML:       installerYaml,
			Component:           name,
			Namespace:           namespace,
			ComponentOrder:      &order,
			ClusterID:           runtimeId,
			GlobalConfiguration: []byte("{}"),
			Configuration:       []byte("{}"),
			Prerequisites:       []byte("{}"),
		}
	}

	for _, testCase := range []struct {
		description    string
		kymaConfigDTO  kymaConfigDTO
		expectedConfig model.KymaConfig
	}{
		{
			description: "should parse using component order",
			kymaConfigDTO: kymaConfigDTO{
				newComponentDTO("comp-3", "even-less-essential", "core", &profileProduction, 3),
				newComponentDTO("comp-1", "essential", "core", &profileProduction, 1),
				newComponentDTO("comp-2", "less-essential", "other", &profileProduction, 2),
			},
			expectedConfig: model.KymaConfig{
				ID: kymaConfigId,
				Release: model.Release{
					Id:            releaseId,
					Version:       version,
					TillerYAML:    tillerYaml,
					InstallerYAML: installerYaml,
				},
				Profile: &expectedProfileProduction,
				Components: []model.KymaComponentConfig{
					{
						ID:             "comp-1",
						Component:      "essential",
						Namespace:      "core",
						SourceURL:      nil,
						Configuration:  model.Configuration{},
						ComponentOrder: 1,
						KymaConfigID:   kymaConfigId,
					},
					{
						ID:             "comp-2",
						Component:      "less-essential",
						Namespace:      "other",
						SourceURL:      nil,
						Configuration:  model.Configuration{},
						ComponentOrder: 2,
						KymaConfigID:   kymaConfigId,
					},
					{
						ID:             "comp-3",
						Component:      "even-less-essential",
						Namespace:      "core",
						SourceURL:      nil,
						Configuration:  model.Configuration{},
						ComponentOrder: 3,
						KymaConfigID:   kymaConfigId,
					},
				},
				GlobalConfiguration: model.Configuration{},
				ClusterID:           runtimeId,
			},
		},
		{
			description: "should parse in order of reed if component order is equal",
			kymaConfigDTO: kymaConfigDTO{
				newComponentDTO("comp-3", "even-less-essential", "core", nil, 0),
				newComponentDTO("comp-1", "essential", "core", nil, 0),
				newComponentDTO("comp-2", "less-essential", "other", nil, 0),
			},
			expectedConfig: model.KymaConfig{
				ID: kymaConfigId,
				Release: model.Release{
					Id:            releaseId,
					Version:       version,
					TillerYAML:    tillerYaml,
					InstallerYAML: installerYaml,
				},
				Components: []model.KymaComponentConfig{
					{
						ID:             "comp-3",
						KymaConfigID:   kymaConfigId,
						Component:      "even-less-essential",
						Namespace:      "core",
						SourceURL:      nil,
						ComponentOrder: 0,
						Prerequisites:  model.Prerequisites{},
						Configuration:  model.Configuration{},
					},
					{
						ID:             "comp-1",
						KymaConfigID:   kymaConfigId,
						Component:      "essential",
						Namespace:      "core",
						SourceURL:      nil,
						ComponentOrder: 0,
						Prerequisites:  model.Prerequisites{},
						Configuration:  model.Configuration{},
					},
					{
						ID:             "comp-2",
						KymaConfigID:   kymaConfigId,
						Component:      "less-essential",
						Namespace:      "other",
						SourceURL:      nil,
						ComponentOrder: 0,
						Prerequisites:  model.Prerequisites{},
						Configuration:  model.Configuration{},
					},
				},
				GlobalConfiguration: model.Configuration{},
				ClusterID:           runtimeId,
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			kymaConfig, err := testCase.kymaConfigDTO.parseToKymaConfig(runtimeId)
			require.NoError(t, err)

			assert.Equal(t, testCase.expectedConfig, kymaConfig)
		})
	}

}
