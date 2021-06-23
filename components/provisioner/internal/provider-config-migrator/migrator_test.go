package migrator

import (
	"encoding/json"
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/stretchr/testify/require"
)

func TestProviderConfigMigrator_Do(t *testing.T) {
	t.Run("should migrate config", func(t *testing.T) {
		//given
		factory := &mocks.Factory{}
		readWriteSession := &mocks.ReadWriteSession{}
		factory.On("NewReadWriteSession").Return(readWriteSession)

		id := "test-id"
		clusterId := "test-cluster-id"
		config := model.SingleZoneAWSProviderConfigInput{
			InternalCidr: "int-cidr",
			PublicCidr:   "pub-cidr",
			VpcCidr:      "vpc",
			Zone:         "aws-zone",
		}

		testData := testProviderData(t, id, clusterId, config)

		expectedProviderConfig := toExpectedConfig(t, testData, config)

		readWriteSession.On("GetProviderSpecificConfigsByProvider", model.AWS).Return([]dbsession.ProviderData{testData}, nil)
		readWriteSession.On("UpdateProviderSpecificConfig", id, expectedProviderConfig).Return(nil)

		migrationJob := NewProviderConfigMigrator(factory, 1)

		err := migrationJob.Do()

		require.NoError(t, err)

		readWriteSession.AssertExpectations(t)
	})

	t.Run("should return error when failed to get configs from database", func(t *testing.T) {
		//given
		factory := &mocks.Factory{}
		readWriteSession := &mocks.ReadWriteSession{}
		factory.On("NewReadWriteSession").Return(readWriteSession)

		readWriteSession.On("GetProviderSpecificConfigsByProvider", model.AWS).Return(nil, dberrors.Internal("error"))

		migrationJob := NewProviderConfigMigrator(factory, 1)

		err := migrationJob.Do()

		require.Error(t, err)

		readWriteSession.AssertExpectations(t)
	})

	t.Run("should return error when exceeded errors threshold", func(t *testing.T) {
		//given
		errorsThreshold := 0

		factory := &mocks.Factory{}
		readWriteSession := &mocks.ReadWriteSession{}
		factory.On("NewReadWriteSession").Return(readWriteSession)

		id := "test-id"
		clusterId := "test-cluster-id"
		config := model.SingleZoneAWSProviderConfigInput{
			InternalCidr: "int-cidr",
			PublicCidr:   "pub-cidr",
			VpcCidr:      "vpc",
			Zone:         "aws-zone",
		}

		testData := testProviderData(t, id, clusterId, config)

		expectedProviderConfig := toExpectedConfig(t, testData, config)

		readWriteSession.On("GetProviderSpecificConfigsByProvider", model.AWS).Return([]dbsession.ProviderData{testData}, nil)
		readWriteSession.On("UpdateProviderSpecificConfig", id, expectedProviderConfig).Return(dberrors.Internal("error"))

		migrationJob := NewProviderConfigMigrator(factory, errorsThreshold)

		err := migrationJob.Do()

		require.Error(t, err)

		readWriteSession.AssertExpectations(t)
	})
}

func toExpectedConfig(t *testing.T, data dbsession.ProviderData, config model.SingleZoneAWSProviderConfigInput) string {
	expConfig := gqlschema.AWSProviderConfigInput{
		VpcCidr: config.VpcCidr,
		AwsZones: []*gqlschema.AWSZoneInput{
			{
				Name:         config.Zone,
				PublicCidr:   config.PublicCidr,
				InternalCidr: config.InternalCidr,
				WorkerCidr:   data.WorkerCidr,
			},
		},
	}

	jsonConfig, err := json.Marshal(expConfig)
	require.NoError(t, err)

	return string(jsonConfig)
}

func testProviderData(t *testing.T, id, clusterId string, config model.SingleZoneAWSProviderConfigInput) dbsession.ProviderData {
	jsonConfig, err := json.Marshal(config)

	require.NoError(t, err)

	return dbsession.ProviderData{
		Id:                     id,
		ClusterId:              clusterId,
		WorkerCidr:             "cidr",
		ProviderSpecificConfig: string(jsonConfig),
	}
}
