package api_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
		CRDDirectoryPaths: []string{filepath.Join("testdata", "crd")},
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
	auditLogTenant    string
	provisioningInput provisioningInput
	upgradeShootInput gqlschema.UpgradeShootInput
	seed              *gardener_types.Seed
}

type provisioningInput struct {
	config       gqlschema.ClusterConfigInput
	runtimeInput gqlschema.RuntimeInput
}

/*
Testing of these provisioning configs is time-consuming!
Here should be added only happy path test cases where parameter differences actually matter from Resolver's perspective.
Everything else should be tested in appropriate package
*/
func newTestProvisioningConfigs() []testCase {
	return []testCase{
		{name: "Azure on Gardener",
			description:    "Should provision, deprovision a runtime and upgrade shoot on happy path, using correct Azure configuration for Gardener, when zones passed",
			runtimeID:      "1100bb59-9c40-4ebb-b846-7477c4dc5bb4",
			auditLogTenant: "12d68c35-556b-4966-a061-235d4a060929",
			provisioningInput: provisioningInput{
				config: azureGardenerClusterConfigInput("1", "2"),
				runtimeInput: gqlschema.RuntimeInput{
					Name:        "test runtime 2",
					Description: new(string),
				}},
			upgradeShootInput: NewUpgradeShootInput(),
			seed:              seedConfig("az-eu2", "cf.eu20", "azure"),
		},
		{name: "Azure on Gardener seed is empty",
			description:    "Should provision, deprovision a runtime and upgrade shoot on happy path, using correct Azure configuration for Gardener, when seed is empty",
			runtimeID:      "1100bb59-9c40-4ebb-b846-7477c4dc5bb2",
			auditLogTenant: "",
			provisioningInput: provisioningInput{
				config: azureGardenerClusterConfigInputNoSeed(),
				runtimeInput: gqlschema.RuntimeInput{
					Name:        "test runtime 5",
					Description: new(string),
				}},
			upgradeShootInput: NewUpgradeShootInput(),
		},
		{name: "OpenStack on Gardener",
			description:    "Should provision, deprovision a runtime and upgrade shoot on happy path, using correct OpenStack configuration for Gardener",
			runtimeID:      "1100bb59-9c40-4ebb-b846-7477c4dc5bb8",
			auditLogTenant: "e7382275-e835-4549-94e1-3b1101e3a1fa",
			provisioningInput: provisioningInput{
				config: openStackGardenerClusterConfigInput(),
				runtimeInput: gqlschema.RuntimeInput{
					Name:        "test runtime 6",
					Description: new(string),
				}},
			upgradeShootInput: NewUpgradeOpenStackShootInput(),
			seed:              seedConfig("os-eu1", "cf.eu10", "openstack"),
		},
	}
}

func azureGardenerClusterConfigInput(zones ...string) gqlschema.ClusterConfigInput {
	return gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			Name:              util.CreateGardenerClusterName(),
			KubernetesVersion: "version",
			Purpose:           util.StringPtr("evaluation"),
			Provider:          "Azure",
			TargetSecret:      "secret",
			Seed:              util.StringPtr("az-eu2"),
			Region:            "westeurope",
			MachineType:       "Standard_D8_v3",
			DiskType:          util.StringPtr("Standard_LRS"),
			VolumeSizeGb:      util.IntPtr(40),
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
			OidcConfig: oidcInput(),
		},
	}
}

func azureGardenerClusterConfigInputNoSeed(zones ...string) gqlschema.ClusterConfigInput {
	return gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			Name:              util.CreateGardenerClusterName(),
			KubernetesVersion: "version",
			Purpose:           util.StringPtr("evaluation"),
			Provider:          "Azure",
			TargetSecret:      "secret",
			Region:            "westeurope",
			MachineType:       "Standard_D8_v3",
			DiskType:          util.StringPtr("Standard_LRS"),
			VolumeSizeGb:      util.IntPtr(40),
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
			OidcConfig: oidcInput(),
		},
	}
}

func openStackGardenerClusterConfigInput() gqlschema.ClusterConfigInput {
	return gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			Name:              util.CreateGardenerClusterName(),
			KubernetesVersion: "version",
			Purpose:           util.StringPtr("evaluation"),
			Provider:          "Openstack",
			TargetSecret:      "secret",
			Seed:              util.StringPtr("os-eu1"),
			Region:            "eu-central-1",
			MachineType:       "t3-xlarge",
			WorkerCidr:        "cidr",
			AutoScalerMin:     1,
			AutoScalerMax:     5,
			MaxSurge:          1,
			MaxUnavailable:    2,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				OpenStackConfig: &gqlschema.OpenStackProviderConfigInput{
					Zones:                []string{"eu-de-1a"},
					FloatingPoolName:     "FloatingIP-external-cp",
					CloudProfileName:     "converged-cloud-cp",
					LoadBalancerProvider: "f5",
				},
			},
			OidcConfig: oidcInput(),
		},
	}
}

func seedConfig(seedName, auditIdentifier, provider string) *gardener_types.Seed {
	return &gardener_types.Seed{
		ObjectMeta: v1.ObjectMeta{
			Name: seedName,
		},
		Spec: gardener_types.SeedSpec{
			Provider: gardener_types.SeedProvider{
				Type: provider,
			}},
		Status: gardener_types.SeedStatus{Conditions: []gardener_types.Condition{
			{Type: "AuditlogServiceAvailability",
				Message: fmt.Sprintf("Auditlog landscape https://api.auditlog.%s.hana.ondemand.com:8081/ successfully attached to the seed.", auditIdentifier),
			},
		}},
	}
}

func NewUpgradeShootInput() gqlschema.UpgradeShootInput {
	return gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion: util.StringPtr("version2"),
			Purpose:           util.StringPtr("testing"),
			MachineType:       util.StringPtr("new-machine"),
			DiskType:          util.StringPtr("papyrus"),
			VolumeSizeGb:      util.IntPtr(50),
			AutoScalerMin:     util.IntPtr(2),
			AutoScalerMax:     util.IntPtr(6),
			MaxSurge:          util.IntPtr(2),
			MaxUnavailable:    util.IntPtr(1),
			OidcConfig:        oidcInput(),
		},
	}
}

func NewUpgradeOpenStackShootInput() gqlschema.UpgradeShootInput {
	return gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion: util.StringPtr("version2"),
			Purpose:           util.StringPtr("testing"),
			MachineType:       util.StringPtr("new-machine"),
			AutoScalerMin:     util.IntPtr(2),
			AutoScalerMax:     util.IntPtr(6),
			MaxSurge:          util.IntPtr(2),
			MaxUnavailable:    util.IntPtr(1),
			OidcConfig:        oidcInput(),
		},
	}
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

	installer := gqlschema.KymaInstallationMethodKymaOperator
	return &gqlschema.KymaConfig{
		Version:       util.StringPtr(kymaVersion),
		KymaInstaller: &installer,
		Components: []*gqlschema.ComponentConfiguration{
			{
				Component:     clusterEssentialsComponent,
				Namespace:     kymaSystemNamespace,
				Prerequisite:  util.BoolPtr(false),
				Configuration: make([]*gqlschema.ConfigEntry, 0, 0),
			},
			{
				Component:     rafterComponent,
				Namespace:     kymaSystemNamespace,
				Prerequisite:  util.BoolPtr(false),
				Configuration: make([]*gqlschema.ConfigEntry, 0, 0),
				SourceURL:     util.StringPtr(rafterSourceURL),
			},
			{
				Component:    coreComponent,
				Namespace:    kymaSystemNamespace,
				Prerequisite: util.BoolPtr(false),
				Configuration: []*gqlschema.ConfigEntry{
					fixGQLConfigEntry("test.config.key", "value", util.BoolPtr(false)),
					fixGQLConfigEntry("test.config.key2", "value2", util.BoolPtr(false)),
				},
			},
			{
				Component:    applicationConnectorComponent,
				Namespace:    kymaIntegrationNamespace,
				Prerequisite: util.BoolPtr(false),
				Configuration: []*gqlschema.ConfigEntry{
					fixGQLConfigEntry("test.config.key", "value", util.BoolPtr(false)),
					fixGQLConfigEntry("test.secret.key", "secretValue", util.BoolPtr(true)),
				},
			},
			{
				Component:    runtimeAgentComponent,
				Namespace:    compassSystemNamespace,
				Prerequisite: util.BoolPtr(false),
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
