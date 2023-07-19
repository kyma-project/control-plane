package api_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/gardener"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	auditLogConfig    *gardener.AuditLogConfig
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
			description: "Should provision, deprovision a runtime and upgrade shoot on happy path, using correct Azure configuration for Gardener, when zones passed",
			runtimeID:   "1100bb59-9c40-4ebb-b846-7477c4dc5bb4",
			auditLogConfig: &gardener.AuditLogConfig{
				TenantID:   "12d68c35-556b-4966-a061-235d4a060929",
				ServiceURL: "https://auditlog.example.com:3001",
				SecretName: "auditlog-secret2",
			},
			provisioningInput: provisioningInput{
				config: azureGardenerClusterConfigInput("1", "2"),
				runtimeInput: gqlschema.RuntimeInput{
					Name:        "test runtime 2",
					Description: new(string),
				}},
			upgradeShootInput: NewUpgradeShootInput(),
			seed:              seedConfig("az-eu2", "eu-west-1", "azure"),
		},
		{name: "Azure on Gardener seed is empty",
			description:    "Should provision, deprovision a runtime and upgrade shoot on happy path, using correct Azure configuration for Gardener, when seed is empty",
			runtimeID:      "1100bb59-9c40-4ebb-b846-7477c4dc5bb2",
			auditLogConfig: nil,
			provisioningInput: provisioningInput{
				config: azureGardenerClusterConfigInputNoSeed(),
				runtimeInput: gqlschema.RuntimeInput{
					Name:        "test runtime 5",
					Description: new(string),
				}},
			upgradeShootInput: NewUpgradeShootInput(),
		},
		{name: "OpenStack on Gardener",
			description: "Should provision, deprovision a runtime and upgrade shoot on happy path, using correct OpenStack configuration for Gardener",
			runtimeID:   "1100bb59-9c40-4ebb-b846-7477c4dc5bb8",
			auditLogConfig: &gardener.AuditLogConfig{
				TenantID:   "e7382275-e835-4549-94e1-3b1101e3a1fa",
				ServiceURL: "https://auditlog.example.com:3000",
				SecretName: "auditlog-secret",
			},
			provisioningInput: provisioningInput{
				config: openStackGardenerClusterConfigInput(),
				runtimeInput: gqlschema.RuntimeInput{
					Name:        "test runtime 6",
					Description: new(string),
				}},
			upgradeShootInput: NewUpgradeOpenStackShootInput(),
			seed:              seedConfig("os-eu1", "eu-central-1", "openstack"),
		},
	}
}

func azureGardenerClusterConfigInput(zones ...string) gqlschema.ClusterConfigInput {
	return gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			Name:                util.CreateGardenerClusterName(),
			KubernetesVersion:   "1.20.7",
			Purpose:             util.StringPtr("evaluation"),
			Provider:            "Azure",
			TargetSecret:        "secret",
			Seed:                util.StringPtr("az-eu2"),
			Region:              "westeurope",
			MachineType:         "Standard_D8_v3",
			MachineImage:        util.StringPtr("red-hat"),
			MachineImageVersion: util.StringPtr("8.0"),
			DiskType:            util.StringPtr("Standard_LRS"),
			VolumeSizeGb:        util.IntPtr(40),
			WorkerCidr:          "cidr",
			AutoScalerMin:       1,
			AutoScalerMax:       5,
			MaxSurge:            1,
			MaxUnavailable:      2,
			ExposureClassName:   util.StringPtr("exp-class"),
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
			Name:                util.CreateGardenerClusterName(),
			KubernetesVersion:   "1.20.7",
			Purpose:             util.StringPtr("evaluation"),
			Provider:            "Azure",
			TargetSecret:        "secret",
			Region:              "westeurope",
			MachineType:         "Standard_D8_v3",
			MachineImage:        util.StringPtr("red-hat"),
			MachineImageVersion: util.StringPtr("8.0"),
			DiskType:            util.StringPtr("Standard_LRS"),
			VolumeSizeGb:        util.IntPtr(40),
			WorkerCidr:          "cidr",
			AutoScalerMin:       1,
			AutoScalerMax:       5,
			MaxSurge:            1,
			MaxUnavailable:      2,
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
			Name:                util.CreateGardenerClusterName(),
			KubernetesVersion:   "1.20.7",
			Purpose:             util.StringPtr("evaluation"),
			Provider:            "Openstack",
			TargetSecret:        "secret",
			Seed:                util.StringPtr("os-eu1"),
			Region:              "eu-central-1",
			MachineType:         "t3-xlarge",
			MachineImage:        util.StringPtr("red-hat"),
			MachineImageVersion: util.StringPtr("8.0"),
			WorkerCidr:          "cidr",
			AutoScalerMin:       1,
			AutoScalerMax:       5,
			MaxSurge:            1,
			MaxUnavailable:      2,
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

func seedConfig(seedName, region, provider string) *gardener_types.Seed {
	return &gardener_types.Seed{
		ObjectMeta: v1.ObjectMeta{
			Name: seedName,
		},
		Spec: gardener_types.SeedSpec{
			Provider: gardener_types.SeedProvider{
				Type:   provider,
				Region: region,
			}},
		Status: gardener_types.SeedStatus{Conditions: []gardener_types.Condition{}},
	}
}

func NewUpgradeShootInput() gqlschema.UpgradeShootInput {
	return gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion:                   util.StringPtr("1.20.8"),
			Purpose:                             util.StringPtr("testing"),
			MachineType:                         util.StringPtr("new-machine"),
			DiskType:                            util.StringPtr("papyrus"),
			VolumeSizeGb:                        util.IntPtr(50),
			AutoScalerMin:                       util.IntPtr(2),
			AutoScalerMax:                       util.IntPtr(6),
			MachineImage:                        util.StringPtr("ubuntu"),
			MachineImageVersion:                 util.StringPtr("12.0.2"),
			MaxSurge:                            util.IntPtr(2),
			MaxUnavailable:                      util.IntPtr(1),
			EnableKubernetesVersionAutoUpdate:   util.BoolPtr(true),
			EnableMachineImageVersionAutoUpdate: util.BoolPtr(true),
			OidcConfig:                          oidcInput(),
			ExposureClassName:                   util.StringPtr("new-exp-class"),
		},
	}
}

func NewUpgradeOpenStackShootInput() gqlschema.UpgradeShootInput {
	return gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion:                   util.StringPtr("1.20.8"),
			Purpose:                             util.StringPtr("testing"),
			MachineType:                         util.StringPtr("new-machine"),
			MachineImage:                        util.StringPtr("ubuntu"),
			MachineImageVersion:                 util.StringPtr("12.0.2"),
			AutoScalerMin:                       util.IntPtr(2),
			AutoScalerMax:                       util.IntPtr(6),
			MaxSurge:                            util.IntPtr(2),
			MaxUnavailable:                      util.IntPtr(1),
			EnableKubernetesVersionAutoUpdate:   util.BoolPtr(true),
			EnableMachineImageVersionAutoUpdate: util.BoolPtr(true),
			OidcConfig:                          oidcInput(),
		},
	}
}
