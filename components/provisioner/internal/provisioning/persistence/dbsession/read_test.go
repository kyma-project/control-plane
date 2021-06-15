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
	kymaOperatorInstaller := string(model.KymaOperatorInstaller)
	parallelInstaller := string(model.ParallelInstaller)

	newComponentDTO := func(id, name, namespace, installer string, profile *string, order int, prerequisite bool) kymaComponentConfigDTO {
		return kymaComponentConfigDTO{
			ID:                  id,
			KymaConfigID:        kymaConfigId,
			ReleaseID:           releaseId,
			Profile:             profile,
			Installer:           installer,
			Version:             version,
			TillerYAML:          tillerYaml,
			InstallerYAML:       installerYaml,
			Component:           name,
			Namespace:           namespace,
			ComponentOrder:      &order,
			ClusterID:           runtimeId,
			GlobalConfiguration: []byte("{}"),
			Configuration:       []byte("{}"),
			Prerequisite:        prerequisite,
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
				newComponentDTO("comp-3", "even-less-essential", "core", kymaOperatorInstaller, &profileProduction, 3, false),
				newComponentDTO("comp-1", "essential", "core", kymaOperatorInstaller, &profileProduction, 1, true),
				newComponentDTO("comp-2", "less-essential", "other", kymaOperatorInstaller, &profileProduction, 2, false),
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
						Prerequisite:   true,
					},
					{
						ID:             "comp-2",
						Component:      "less-essential",
						Namespace:      "other",
						SourceURL:      nil,
						Configuration:  model.Configuration{},
						ComponentOrder: 2,
						KymaConfigID:   kymaConfigId,
						Prerequisite:   false,
					},
					{
						ID:             "comp-3",
						Component:      "even-less-essential",
						Namespace:      "core",
						SourceURL:      nil,
						Configuration:  model.Configuration{},
						ComponentOrder: 3,
						KymaConfigID:   kymaConfigId,
						Prerequisite:   false,
					},
				},
				GlobalConfiguration: model.Configuration{},
				ClusterID:           runtimeId,
				Installer:           model.KymaOperatorInstaller,
			},
		},
		{
			description: "should parse in order of reed if component order is equal",
			kymaConfigDTO: kymaConfigDTO{
				newComponentDTO("comp-3", "even-less-essential", "core", parallelInstaller, nil, 0, false),
				newComponentDTO("comp-1", "essential", "core", parallelInstaller, nil, 0, true),
				newComponentDTO("comp-2", "less-essential", "other", parallelInstaller, nil, 0, false),
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
						Component:      "even-less-essential",
						Namespace:      "core",
						SourceURL:      nil,
						Configuration:  model.Configuration{},
						ComponentOrder: 0,
						KymaConfigID:   kymaConfigId,
						Prerequisite:   false,
					},
					{
						ID:             "comp-1",
						Component:      "essential",
						Namespace:      "core",
						SourceURL:      nil,
						Configuration:  model.Configuration{},
						ComponentOrder: 0,
						KymaConfigID:   kymaConfigId,
						Prerequisite:   true,
					},
					{
						ID:             "comp-2",
						Component:      "less-essential",
						Namespace:      "other",
						SourceURL:      nil,
						Configuration:  model.Configuration{},
						ComponentOrder: 0,
						KymaConfigID:   kymaConfigId,
						Prerequisite:   false,
					},
				},
				GlobalConfiguration: model.Configuration{},
				ClusterID:           runtimeId,
				Installer:           model.ParallelInstaller,
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
