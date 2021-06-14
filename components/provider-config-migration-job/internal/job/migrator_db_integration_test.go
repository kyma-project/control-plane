package job

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/model"
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/persistence/dbconnection"
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/testutils"
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var uuid = testutils.NewUUIDGenerator()

func TestProviderConfigMigrator(t *testing.T) {
	ctx := context.Background()

	cleanupNetwork, err := testutils.EnsureTestNetworkForDB(t, ctx)
	require.NoError(t, err)
	defer cleanupNetwork()

	containerCleanupFunc, connString, err := testutils.InitTestDBContainer(t, ctx, "postgres_database_2")
	require.NoError(t, err)
	defer containerCleanupFunc()

	connection, err := dbconnection.InitializeDatabaseConnection(connString, 5)
	require.NoError(t, err)
	require.NotNil(t, connection)
	defer testutils.CloseDatabase(t, connection)

	err = testutils.SetupSchema(connection, testutils.SchemaFilePath)
	require.NoError(t, err)

	factory := dbconnection.NewFactory(connection)

	release := prepareTestRelease(t, factory)

	providerConfig := createdTestProviderConfigData()

	clusterID := uuid.New()

	gardenerID := uuid.New()
	gardenerConfig := createFixedGardenerConfig(t, providerConfig, clusterID, gardenerID)

	prepareTestRecord(t, factory, release, gardenerConfig, clusterID)

	migrationJob := NewProviderConfigMigrator(factory, 1)

	err = migrationJob.Do()

	require.NoError(t, err)

	newConfig := getUpdatedConfig(t, factory, gardenerID)

	checkIfConfigIsProperlyMigrated(t, newConfig, gardenerConfig, providerConfig)
}

func checkIfConfigIsProperlyMigrated(t *testing.T, config model.AWSProviderConfigInput, gardenerConfig model.GardenerConfig, oldConfig model.OldAWSProviderConfigInput) {
	assert.Equal(t, oldConfig.VpcCidr, config.VpcCidr)
	assert.Equal(t, gardenerConfig.WorkerCidr, config.AwsZones[0].WorkerCidr)
	assert.Equal(t, oldConfig.PublicCidr, config.AwsZones[0].PublicCidr)
	assert.Equal(t, oldConfig.InternalCidr, config.AwsZones[0].InternalCidr)
	assert.Equal(t, oldConfig.Zone, config.AwsZones[0].Name)
}

func getUpdatedConfig(t *testing.T, factory dbconnection.Factory, id string) model.AWSProviderConfigInput {
	session := factory.NewReadWriteSession()
	configJson, dberr := session.GetUpdatedProviderSpecificConfigByID(id)
	require.NoError(t, dberr)

	var configInput model.AWSProviderConfigInput

	err := util.DecodeJson(configJson, &configInput)
	require.NoError(t, err)

	return configInput
}

func prepareTestRelease(t *testing.T, factory dbconnection.Factory) model.Release {
	session := factory.NewReadWriteSession()
	releaseID := uuid.New()
	release := createFixedTestRealease(releaseID)
	err := session.InsertRelease(release)
	require.NoError(t, err)
	return release
}

func prepareTestRecord(t *testing.T, factory dbconnection.Factory, release model.Release, config model.GardenerConfig, clusterID string) {
	session, err := factory.NewSessionWithinTransaction()
	require.NoError(t, err)
	kymaConfigID := uuid.New()
	kymaConfig := createTestKymaConfig(kymaConfigID, clusterID, release)
	cluster := createFixedClusterConfig(clusterID, kymaConfigID)
	err = session.InsertCluster(cluster)
	require.NoError(t, err)
	err = session.InsertGardenerConfig(config)
	require.NoError(t, err)
	err = session.InsertKymaConfig(kymaConfig)
	require.NoError(t, err)
	err = session.Commit()
	require.NoError(t, err)
}

func createFixedClusterConfig(clusterID, kymaConfigID string) model.Cluster {
	return model.Cluster{
		ID:                clusterID,
		CreationTimestamp: time.Time{},
		Tenant:            "tenant",
		SubAccountId:      util.StringPtr("subaccount"),
		KymaConfigID:      kymaConfigID,
	}
}

func createFixedGardenerConfig(t *testing.T, providerSpecConfig model.OldAWSProviderConfigInput, clusterID, gardenerID string) model.GardenerConfig {
	config, err := json.Marshal(providerSpecConfig)

	require.NoError(t, err)

	return model.GardenerConfig{
		ID:                                  gardenerID,
		ClusterID:                           clusterID,
		Name:                                "",
		ProjectName:                         "frog-dev",
		KubernetesVersion:                   "1.18",
		VolumeSizeGB:                        util.IntPtr(32),
		DiskType:                            util.StringPtr("disk"),
		MachineType:                         "super-fast",
		MachineImage:                        nil,
		MachineImageVersion:                 nil,
		Provider:                            "aws",
		Purpose:                             util.StringPtr("eval"),
		LicenceType:                         util.StringPtr("license"),
		Seed:                                "aws-seed",
		TargetSecret:                        "aws-secret",
		Region:                              "aws-region",
		WorkerCidr:                          "cidr",
		AutoScalerMin:                       1,
		AutoScalerMax:                       3,
		MaxSurge:                            2,
		MaxUnavailable:                      1,
		EnableKubernetesVersionAutoUpdate:   false,
		EnableMachineImageVersionAutoUpdate: false,
		AllowPrivilegedContainers:           false,
		GardenerProviderConfig:              string(config),
	}
}

func createdTestProviderConfigData() model.OldAWSProviderConfigInput {
	return model.OldAWSProviderConfigInput{
		InternalCidr: "int-cidr",
		PublicCidr:   "pub-cidr",
		VpcCidr:      "vpc",
		Zone:         "aws-zone",
	}
}

func createFixedTestRealease(releaseID string) model.Release {
	return model.Release{
		Id:            releaseID,
		Version:       "2.0",
		TillerYAML:    "something",
		InstallerYAML: "something",
	}
}

func createTestKymaConfig(kymaConfigID, clusterID string, release model.Release) model.KymaConfig {
	return model.KymaConfig{
		ID:        kymaConfigID,
		Release:   release,
		Profile:   model.ProductionProfile,
		ClusterID: clusterID,
		Active:    true,
	}
}
