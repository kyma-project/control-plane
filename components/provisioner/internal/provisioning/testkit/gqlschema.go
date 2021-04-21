package testkit

import (
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

func FixKymaGraphQLConfig(profile *gqlschema.KymaProfile) *gqlschema.KymaConfig {
	return &gqlschema.KymaConfig{
		Version: util.StringPtr(KymaVersion),
		Profile: profile,
		Components: []*gqlschema.ComponentConfiguration{
			{
				Component:     ClusterEssentialsComponent,
				Namespace:     KymaSystemNamespace,
				Prerequisite:  util.BoolPtr(false),
				Configuration: make([]*gqlschema.ConfigEntry, 0, 0),
			},
			{
				Component:    CoreComponent,
				Namespace:    KymaSystemNamespace,
				Prerequisite: util.BoolPtr(false),
				Configuration: []*gqlschema.ConfigEntry{
					FixGQLConfigEntry("test.config.key1", "value1", util.BoolPtr(false)),
					FixGQLConfigEntry("test.config.key2", "value2", util.BoolPtr(false)),
				},
			},
			{
				Component:     RafterComponent,
				Namespace:     KymaSystemNamespace,
				SourceURL:     util.StringPtr(RafterSourceURL),
				Prerequisite:  util.BoolPtr(false),
				Configuration: make([]*gqlschema.ConfigEntry, 0, 0),
			},
			{
				Component: ApplicationConnectorComponent,
				Namespace: KymaIntegrationNamespace,
				Configuration: []*gqlschema.ConfigEntry{
					FixGQLConfigEntry("test.config.key", "value", util.BoolPtr(false)),
					FixGQLConfigEntry("test.secret.key", "secretValue", util.BoolPtr(true)),
				},
				Prerequisite: util.BoolPtr(true),
				PrerequisiteResources: &gqlschema.PrerequisiteResources{
					Secrets: []*gqlschema.SecretPrerequisite{
						FixGQLSecretPrerequisite("prerequisite-secret1",
							FixGQLSecretPrerequisiteEntry("key1", "value1"),
							FixGQLSecretPrerequisiteEntry("key2", "value2"),
						),
						FixGQLSecretPrerequisite("prerequisite-secret2",
							FixGQLSecretPrerequisiteEntry("key1", "value1"),
							FixGQLSecretPrerequisiteEntry("key2", "value2"),
						),
					},
					Certificates: []*gqlschema.GardenerCertificatePrerequisite{
						FixGQLGardenerCertificatePrerequisite("certificate", "secret", "domain.com"),
					},
				},
			},
		},
		Configuration: []*gqlschema.ConfigEntry{
			FixGQLConfigEntry("global.config.key1", "globalValue1", util.BoolPtr(false)),
			FixGQLConfigEntry("global.config.key2", "globalValue2", util.BoolPtr(false)),
			FixGQLConfigEntry("global.secret.key1", "globalSecretValue1", util.BoolPtr(true)),
		},
	}
}

func FixGQLSecretPrerequisiteEntry(key, val string) *gqlschema.SecretPrerequisiteEntry {
	return &gqlschema.SecretPrerequisiteEntry{
		Key:   key,
		Value: val,
	}
}

func FixGQLGardenerCertificatePrerequisite(resource, secret, common string) *gqlschema.GardenerCertificatePrerequisite {
	return &gqlschema.GardenerCertificatePrerequisite{
		ResourceName: resource,
		SecretName:   secret,
		CommonName:   common,
	}
}

func FixGQLSecretPrerequisite(name string, entries ...*gqlschema.SecretPrerequisiteEntry) *gqlschema.SecretPrerequisite {
	return &gqlschema.SecretPrerequisite{
		ResourceName: name,
		Entries:      entries,
	}
}

func FixGQLConfigEntry(key, val string, secret *bool) *gqlschema.ConfigEntry {
	return &gqlschema.ConfigEntry{
		Key:    key,
		Value:  val,
		Secret: secret,
	}
}

func FixGQLSecretPrerequisiteInput(name string, entries ...*gqlschema.SecretPrerequisiteEntryInput) *gqlschema.SecretPrerequisiteInput {
	return &gqlschema.SecretPrerequisiteInput{
		ResourceName: name,
		Entries:      entries,
	}
}

func FixGQLSecretPrerequisiteEntryInput(key, val string) *gqlschema.SecretPrerequisiteEntryInput {
	return &gqlschema.SecretPrerequisiteEntryInput{
		Key:   key,
		Value: val,
	}
}

func FixGQLGardenerCertificatePrerequisiteInput(resourceName, secretName, commonName string) *gqlschema.GardenerCertificatePrerequisiteInput {
	return &gqlschema.GardenerCertificatePrerequisiteInput{
		ResourceName: resourceName,
		SecretName:   secretName,
		CommonName:   commonName,
	}
}
