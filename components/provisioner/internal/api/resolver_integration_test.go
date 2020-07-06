package api_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/installation/release"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestMain(m *testing.M) {
	err := setupEnv()
	if err != nil {
		logrus.Errorf("Failed to setup test environment: %s", err.Error())
		os.Exit(1)
	}
	defer func() {
		err := testEnv.Stop()
		if err != nil {
			logrus.Errorf("error while deleting Compass Connection: %s", err.Error())
		}
	}()

	syncPeriod := syncPeriod

	mgr, err = ctrl.NewManager(cfg, ctrl.Options{SyncPeriod: &syncPeriod, Namespace: namespace})

	if err != nil {
		logrus.Errorf("unable to create shoot controller mgr: %s", err.Error())
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func setupEnv() error {
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("testdata")},
	}

	var err error
	cfg, err = testEnv.Start()
	if err != nil {
		return errors.Wrap(err, "Failed to start test environment")
	}

	return nil
}

type testCase struct {
	name              string
	description       string
	runtimeID         string
	provisioningInput provisioningInput
	upgradeShootInput gqlschema.UpgradeShootInput
}

type provisioningInput struct {
	config       gqlschema.ClusterConfigInput
	runtimeInput gqlschema.RuntimeInput
}

func newTestProvisioningConfigs() []testCase {
	return []testCase{
		{name: "GCP on Gardener",
			description: "Should provision, deprovision a runtime and upgrade shoot on happy path, using correct GCP configuration for Gardener",
			runtimeID:   "1100bb59-9c40-4ebb-b846-7477c4dc5bbb",
			provisioningInput: provisioningInput{
				config: gcpGardenerClusterConfigInput(),
				runtimeInput: gqlschema.RuntimeInput{
					Name:        "test runtime 1",
					Description: new(string),
				}},
			upgradeShootInput: newUpgradeShootInput(),
		},
		{name: "Azure on Gardener (with zones)",
			description: "Should provision, deprovision a runtime and upgrade shoot on happy path, using correct Azure configuration for Gardener, when zones passed",
			runtimeID:   "1100bb59-9c40-4ebb-b846-7477c4dc5bb4",
			provisioningInput: provisioningInput{
				config: azureGardenerClusterConfigInput("fix-az-zone-1", "fix-az-zone-2"),
				runtimeInput: gqlschema.RuntimeInput{
					Name:        "test runtime 2",
					Description: new(string),
				}},
			upgradeShootInput: newAzureUpgradeShootInput(),
		},
		{name: "Azure on Gardener (without zones)",
			description: "Should provision, deprovision a runtime and upgrade shoot on happy path, using correct Azure configuration for Gardener, when zones are empty",
			runtimeID:   "1100bb59-9c40-4ebb-b846-7477c4dc5bb1",
			provisioningInput: provisioningInput{
				config: azureGardenerClusterConfigInput(),
				runtimeInput: gqlschema.RuntimeInput{
					Name:        "test runtime 3",
					Description: new(string),
				}},
			upgradeShootInput: newAzureUpgradeShootInput(),
		},
		{name: "AWS on Gardener",
			description: "Should provision, deprovision a runtime and upgrade shoot on happy path, using correct AWS configuration for Gardener",
			runtimeID:   "1100bb59-9c40-4ebb-b846-7477c4dc5bb5",
			provisioningInput: provisioningInput{
				config: awsGardenerClusterConfigInput(),
				runtimeInput: gqlschema.RuntimeInput{
					Name:        "test runtime4",
					Description: new(string),
				}},
			upgradeShootInput: newUpgradeShootInput(),
		},
	}
}

func gcpGardenerClusterConfigInput() gqlschema.ClusterConfigInput {
	return gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			KubernetesVersion: "version",
			Provider:          "GCP",
			TargetSecret:      "secret",
			Seed:              util.StringPtr("gcp-eu1"),
			Region:            "europe-west1",
			MachineType:       "n1-standard-1",
			DiskType:          "pd-ssd",
			VolumeSizeGb:      40,
			WorkerCidr:        "cidr",
			AutoScalerMin:     1,
			AutoScalerMax:     5,
			MaxSurge:          1,
			MaxUnavailable:    2,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				GcpConfig: &gqlschema.GCPProviderConfigInput{
					Zones: []string{"fix-gcp-zone1", "fix-gcp-zone-2"},
				},
			},
		},
	}
}

func azureGardenerClusterConfigInput(zones ...string) gqlschema.ClusterConfigInput {
	return gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			KubernetesVersion: "version",
			Provider:          "Azure",
			TargetSecret:      "secret",
			Seed:              util.StringPtr("az-eu1"),
			Region:            "westeurope",
			MachineType:       "Standard_D8_v3",
			DiskType:          "Standard_LRS",
			VolumeSizeGb:      40,
			WorkerCidr:        "cidr",
			AutoScalerMin:     1,
			AutoScalerMax:     5,
			MaxSurge:          1,
			MaxUnavailable:    2,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AzureConfig: &gqlschema.AzureProviderConfigInput{
					VnetCidr: "cidr",
					Zones:    zones,
				},
			},
		},
	}
}

func awsGardenerClusterConfigInput() gqlschema.ClusterConfigInput {
	return gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			KubernetesVersion: "version",
			Provider:          "AWS",
			TargetSecret:      "secret",
			Seed:              nil,
			Region:            "eu-central-1",
			MachineType:       "t3-xlarge",
			DiskType:          "gp2",
			VolumeSizeGb:      40,
			WorkerCidr:        "cidr",
			AutoScalerMin:     1,
			AutoScalerMax:     5,
			MaxSurge:          1,
			MaxUnavailable:    2,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				AwsConfig: &gqlschema.AWSProviderConfigInput{
					Zone:         "zone",
					InternalCidr: "cidr",
					VpcCidr:      "cidr",
					PublicCidr:   "cidr",
				},
			},
		},
	}
}

func newUpgradeShootInput() gqlschema.UpgradeShootInput {
	newKubernetesVersion := "version2"
	newMachineType := "new-machine"
	newDiskType := "papyrus"
	newVolumeSizeGb := 50
	newCidr := "cidr2"

	return gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion: &newKubernetesVersion,
			MachineType:       &newMachineType,
			DiskType:          &newDiskType,
			VolumeSizeGb:      &newVolumeSizeGb,
			WorkerCidr:        &newCidr,
			AutoScalerMin:     util.IntPtr(2),
			AutoScalerMax:     util.IntPtr(6),
			MaxSurge:          util.IntPtr(2),
			MaxUnavailable:    util.IntPtr(1),
		},
	}
}

func newAzureUpgradeShootInput() gqlschema.UpgradeShootInput {
	input := newUpgradeShootInput()
	input.GardenerConfig.ProviderSpecificConfig = &gqlschema.ProviderSpecificInput{
		AzureConfig: &gqlschema.AzureProviderConfigInput{
			VnetCidr: "cidr2",
		},
	}
	return input
}

func insertDummyReleaseIfNotExist(releaseRepo release.Repository, id, version string) error {
	_, err := releaseRepo.GetReleaseByVersion(version)
	if err == nil {
		return nil
	}

	if err.Code() != dberrors.CodeNotFound {
		return err
	}
	_, err = releaseRepo.SaveRelease(model.Release{
		Id:            id,
		Version:       version,
		TillerYAML:    "tiller YAML",
		InstallerYAML: "installer YAML",
	})

	return err
}

func fixKymaGraphQLConfigInput() *gqlschema.KymaConfigInput {

	return &gqlschema.KymaConfigInput{
		Version: kymaVersion,
		Components: []*gqlschema.ComponentConfigurationInput{
			{
				Component: clusterEssentialsComponent,
				Namespace: kymaSystemNamespace,
			},
			{
				Component: rafterComponent,
				Namespace: kymaSystemNamespace,
				SourceURL: util.StringPtr(rafterSourceURL),
			},
			{
				Component: coreComponent,
				Namespace: kymaSystemNamespace,
				Configuration: []*gqlschema.ConfigEntryInput{
					fixGQLConfigEntryInput("test.config.key", "value", util.BoolPtr(false)),
					fixGQLConfigEntryInput("test.config.key2", "value2", util.BoolPtr(false)),
				},
			},
			{
				Component: applicationConnectorComponent,
				Namespace: kymaIntegrationNamespace,
				Configuration: []*gqlschema.ConfigEntryInput{
					fixGQLConfigEntryInput("test.config.key", "value", util.BoolPtr(false)),
					fixGQLConfigEntryInput("test.secret.key", "secretValue", util.BoolPtr(true)),
				},
			},
			{
				Component: runtimeAgentComponent,
				Namespace: compassSystemNamespace,
				Configuration: []*gqlschema.ConfigEntryInput{
					fixGQLConfigEntryInput("test.config.key", "value", util.BoolPtr(false)),
					fixGQLConfigEntryInput("test.secret.key", "secretValue", util.BoolPtr(true)),
				},
			},
		},
		Configuration: []*gqlschema.ConfigEntryInput{
			fixGQLConfigEntryInput("global.config.key", "globalValue", util.BoolPtr(false)),
			fixGQLConfigEntryInput("global.config.key2", "globalValue2", util.BoolPtr(false)),
			fixGQLConfigEntryInput("global.secret.key", "globalSecretValue", util.BoolPtr(true)),
		},
	}
}

func fixGQLConfigEntryInput(key, val string, secret *bool) *gqlschema.ConfigEntryInput {
	return &gqlschema.ConfigEntryInput{
		Key:    key,
		Value:  val,
		Secret: secret,
	}
}

func fixKymaGraphQLConfig() *gqlschema.KymaConfig {

	return &gqlschema.KymaConfig{
		Version: util.StringPtr(kymaVersion),
		Components: []*gqlschema.ComponentConfiguration{
			{
				Component:     clusterEssentialsComponent,
				Namespace:     kymaSystemNamespace,
				Configuration: make([]*gqlschema.ConfigEntry, 0, 0),
			},
			{
				Component:     rafterComponent,
				Namespace:     kymaSystemNamespace,
				Configuration: make([]*gqlschema.ConfigEntry, 0, 0),
				SourceURL:     util.StringPtr(rafterSourceURL),
			},
			{
				Component: coreComponent,
				Namespace: kymaSystemNamespace,
				Configuration: []*gqlschema.ConfigEntry{
					fixGQLConfigEntry("test.config.key", "value", util.BoolPtr(false)),
					fixGQLConfigEntry("test.config.key2", "value2", util.BoolPtr(false)),
				},
			},
			{
				Component: applicationConnectorComponent,
				Namespace: kymaIntegrationNamespace,
				Configuration: []*gqlschema.ConfigEntry{
					fixGQLConfigEntry("test.config.key", "value", util.BoolPtr(false)),
					fixGQLConfigEntry("test.secret.key", "secretValue", util.BoolPtr(true)),
				},
			},
			{
				Component: runtimeAgentComponent,
				Namespace: compassSystemNamespace,
				Configuration: []*gqlschema.ConfigEntry{
					fixGQLConfigEntry("test.config.key", "value", util.BoolPtr(false)),
					fixGQLConfigEntry("test.secret.key", "secretValue", util.BoolPtr(true)),
				},
			},
		},
		Configuration: []*gqlschema.ConfigEntry{
			fixGQLConfigEntry("global.config.key", "globalValue", util.BoolPtr(false)),
			fixGQLConfigEntry("global.config.key2", "globalValue2", util.BoolPtr(false)),
			fixGQLConfigEntry("global.secret.key", "globalSecretValue", util.BoolPtr(true)),
		},
	}
}

func fixGQLConfigEntry(key, val string, secret *bool) *gqlschema.ConfigEntry {
	return &gqlschema.ConfigEntry{
		Key:    key,
		Value:  val,
		Secret: secret,
	}
}
