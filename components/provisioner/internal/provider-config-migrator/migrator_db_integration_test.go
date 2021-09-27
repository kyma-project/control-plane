package migrator

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/database"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/testutils"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/internal/uuid"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var uuidGenerator = uuid.NewUUIDGenerator()

func TestProviderConfigMigrator(t *testing.T) {
	ctx := context.Background()

	cleanupNetwork, err := testutils.EnsureTestNetworkForDB(t, ctx)
	require.NoError(t, err)
	defer cleanupNetwork()

	containerCleanupFunc, connString, err := testutils.InitTestDBContainer(t, ctx, "postgres_database_2")
	require.NoError(t, err)
	defer containerCleanupFunc()

	connection, err := database.InitializeDatabaseConnection(connString, 5)
	require.NoError(t, err)
	require.NotNil(t, connection)
	defer testutils.CloseDatabase(t, connection)

	err = database.SetupSchema(connection, testutils.SchemaFilePath)
	require.NoError(t, err)

	factory := dbsession.NewFactory(connection)

	release := prepareTestRelease(t, factory)

	providerConfig := createdTestProviderConfigData()

	clusterID := uuidGenerator.New()

	gardenerID := uuidGenerator.New()
	gardenerConfig := createFixedGardenerConfig(t, providerConfig, clusterID, gardenerID)

	prepareTestRecord(t, factory, release, gardenerConfig, clusterID)

	migrationJob := NewProviderConfigMigrator(factory, 1)

	err = migrationJob.Do()

	require.NoError(t, err)

	newConfig := getUpdatedConfig(t, factory, gardenerID)

	checkIfConfigIsProperlyMigrated(t, newConfig, gardenerConfig, providerConfig)
}

func checkIfConfigIsProperlyMigrated(t *testing.T, config gqlschema.AWSProviderConfigInput, gardenerConfig model.GardenerConfig, oldConfig model.SingleZoneAWSProviderConfigInput) {
	assert.Equal(t, oldConfig.VpcCidr, config.VpcCidr)
	assert.Equal(t, gardenerConfig.WorkerCidr, config.AwsZones[0].WorkerCidr)
	assert.Equal(t, oldConfig.PublicCidr, config.AwsZones[0].PublicCidr)
	assert.Equal(t, oldConfig.InternalCidr, config.AwsZones[0].InternalCidr)
	assert.Equal(t, oldConfig.Zone, config.AwsZones[0].Name)
}

func getUpdatedConfig(t *testing.T, factory dbsession.Factory, id string) gqlschema.AWSProviderConfigInput {
	session := factory.NewReadWriteSession()
	configJson, dberr := session.GetUpdatedProviderSpecificConfigByID(id)
	require.NoError(t, dberr)

	var configInput gqlschema.AWSProviderConfigInput

	err := util.DecodeJson(configJson, &configInput)
	require.NoError(t, err)

	return configInput
}

func prepareTestRelease(t *testing.T, factory dbsession.Factory) model.Release {
	session := factory.NewReadWriteSession()
	releaseID := uuidGenerator.New()
	release := createFixedTestRealease(releaseID)
	err := session.InsertRelease(release)
	require.NoError(t, err)
	return release
}

func prepareTestRecord(t *testing.T, factory dbsession.Factory, release model.Release, config model.GardenerConfig, clusterID string) {
	session, err := factory.NewSessionWithinTransaction()
	require.NoError(t, err)
	kymaConfigID := uuidGenerator.New()
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
		KymaConfig:        &model.KymaConfig{ID: kymaConfigID},
	}
}

func createFixedGardenerConfig(t *testing.T, providerSpecConfig model.SingleZoneAWSProviderConfigInput, clusterID, gardenerID string) model.GardenerConfig {
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
		Provider:                            model.AWS,
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
		GardenerProviderConfig: SingleZoneAWSGardenerConfig{
			input:                  &providerSpecConfig,
			ProviderSpecificConfig: model.ProviderSpecificConfig(config),
		},
	}
}

func createdTestProviderConfigData() model.SingleZoneAWSProviderConfigInput {
	return model.SingleZoneAWSProviderConfigInput{
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
	profile := model.EvaluationProfile
	return model.KymaConfig{
		ID:        kymaConfigID,
		Release:   release,
		Profile:   &profile,
		ClusterID: clusterID,
		Active:    true,
	}
}

type SingleZoneAWSGardenerConfig struct {
	model.ProviderSpecificConfig
	input *model.SingleZoneAWSProviderConfigInput `db:"-"`
}

func (c SingleZoneAWSGardenerConfig) AsProviderSpecificConfig() gqlschema.ProviderSpecificConfig {
	return nil
}

func (c SingleZoneAWSGardenerConfig) EditShootConfig(_ model.GardenerConfig, _ *gardener_types.Shoot) apperrors.AppError {
	return nil
}

func (c SingleZoneAWSGardenerConfig) ExtendShootConfig(_ model.GardenerConfig, _ *gardener_types.Shoot) apperrors.AppError {
	return nil
}
