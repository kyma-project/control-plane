package testkit

import (
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
)

func FixKymaConfig(profile *model.KymaProfile) model.KymaConfig {
	return model.KymaConfig{
		ID:                  "id",
		Release:             FixKymaRelease(),
		Profile:             profile,
		Components:          FixKymaComponents(),
		GlobalConfiguration: FixGlobalConfig(),
		ClusterID:           "runtimeID",
	}
}

func FixGlobalConfig() model.Configuration {
	return model.Configuration{
		ConfigEntries: []model.ConfigEntry{
			model.NewConfigEntry("global.config.key1", "globalValue1", false),
			model.NewConfigEntry("global.config.key2", "globalValue2", false),
			model.NewConfigEntry("global.secret.key1", "globalSecretValue1", true),
		},
	}
}

func FixKymaComponents() []model.KymaComponentConfig {
	return []model.KymaComponentConfig{
		{
			ID:             "id",
			KymaConfigID:   "id",
			Component:      ClusterEssentialsComponent,
			Namespace:      KymaSystemNamespace,
			Configuration:  model.Configuration{ConfigEntries: make([]model.ConfigEntry, 0, 0)},
			ComponentOrder: 1,
		},
		{
			ID:           "id",
			KymaConfigID: "id",
			Component:    CoreComponent,
			Namespace:    KymaSystemNamespace,
			Configuration: model.Configuration{
				ConfigEntries: []model.ConfigEntry{
					model.NewConfigEntry("test.config.key1", "value1", false),
					model.NewConfigEntry("test.config.key2", "value2", false),
				},
			},
			ComponentOrder: 2,
		},
		{
			ID:             "id",
			KymaConfigID:   "id",
			Component:      RafterComponent,
			Namespace:      KymaSystemNamespace,
			SourceURL:      util.StringPtr(RafterSourceURL),
			Configuration:  model.Configuration{ConfigEntries: make([]model.ConfigEntry, 0, 0)},
			ComponentOrder: 3,
		},
		{
			ID:           "id",
			KymaConfigID: "id",
			Component:    ApplicationConnectorComponent,
			Namespace:    KymaIntegrationNamespace,
			Configuration: model.Configuration{
				ConfigEntries: []model.ConfigEntry{
					model.NewConfigEntry("test.config.key", "value", false),
					model.NewConfigEntry("test.secret.key", "secretValue", true),
				},
			},
			Prerequisites: model.Prerequisites{
				Secrets: []model.SecretPrerequisite{
					{
						ResourceName: "prerequisite-secret1",
						Entries: []model.SecretEntry{
							model.NewSecretEntry("key1", "value1"),
							model.NewSecretEntry("key2", "value2"),
						},
					},
					{
						ResourceName: "prerequisite-secret2",
						Entries: []model.SecretEntry{
							model.NewSecretEntry("key1", "value1"),
							model.NewSecretEntry("key2", "value2"),
						},
					},
				},
				Certificates: []model.GardenerCertificatePrerequisite{
					model.NewGardenerCertificatePrerequisite("certificate", "secret", "domain.com"),
				},
			},
			ComponentOrder: 4,
		},
	}
}

func FixKymaRelease() model.Release {
	return model.Release{
		Id:            "d829b1b5-2e82-426d-91b0-f94978c0c140",
		Version:       KymaVersion,
		TillerYAML:    "tiller yaml",
		InstallerYAML: "installer yaml",
	}
}

func FixKymaReleaseWithoutTiller() model.Release {
	return model.Release{
		Id:            "e829b1b5-2e82-426d-91b0-f94978c0c140",
		Version:       KymaVersionWithoutTiller,
		TillerYAML:    "",
		InstallerYAML: "installer yaml",
	}
}
