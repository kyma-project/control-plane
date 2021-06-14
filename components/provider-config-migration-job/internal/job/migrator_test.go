package job

import (
	"encoding/json"
	"testing"

	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/model"
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/persistence/dbconnection"
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/persistence/dbconnection/mocks"
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/persistence/dberrors"
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
		config := model.OldAWSProviderConfigInput{
			InternalCidr: "int-cidr",
			PublicCidr:   "pub-cidr",
			VpcCidr:      "vpc",
			Zone:         "aws-zone",
		}

		testData := testProviderData(t, id, clusterId, config)

		expectedProviderConfig := toExpectedConfig(t, testData, config)

		readWriteSession.On("GetProviderSpecificConfigsByProvider", model.AWS).Return([]dbconnection.ProviderData{testData}, nil)
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
		config := model.OldAWSProviderConfigInput{
			InternalCidr: "int-cidr",
			PublicCidr:   "pub-cidr",
			VpcCidr:      "vpc",
			Zone:         "aws-zone",
		}

		testData := testProviderData(t, id, clusterId, config)

		expectedProviderConfig := toExpectedConfig(t, testData, config)

		readWriteSession.On("GetProviderSpecificConfigsByProvider", model.AWS).Return([]dbconnection.ProviderData{testData}, nil)
		readWriteSession.On("UpdateProviderSpecificConfig", id, expectedProviderConfig).Return(dberrors.Internal("error"))

		migrationJob := NewProviderConfigMigrator(factory, errorsThreshold)

		err := migrationJob.Do()

		require.Error(t, err)

		readWriteSession.AssertExpectations(t)
	})
}

func toExpectedConfig(t *testing.T, data dbconnection.ProviderData, config model.OldAWSProviderConfigInput) string {
	expConfig := model.AWSProviderConfigInput{
		VpcCidr: config.VpcCidr,
		AwsZones: []*model.AWSZoneInput{
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

func testProviderData(t *testing.T, id, clusterId string, config model.OldAWSProviderConfigInput) dbconnection.ProviderData {
	jsonConfig, err := json.Marshal(config)

	require.NoError(t, err)

	return dbconnection.ProviderData{
		Id:                     id,
		ClusterId:              clusterId,
		WorkerCidr:             "cidr",
		ProviderSpecificConfig: string(jsonConfig),
	}
}
