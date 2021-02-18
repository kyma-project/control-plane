package broker_test

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	operationID          = "23caac24-c317-47d0-bd2f-6b1bf4bdba99"
	operationDescription = "some operation status description"
	instID               = "c39d9b98-5ed9-4a68-b786-f26ce93a734f"
)

func TestLastOperation_LastOperation(t *testing.T) {
	t.Run("Should return last operation when operation ID provided", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Operations().InsertProvisioningOperation(fixOperation())
		assert.NoError(t, err)

		lastOperationEndpoint := broker.NewLastOperation(memoryStorage.Operations(), memoryStorage.Instances(), logrus.StandardLogger())

		// when
		response, err := lastOperationEndpoint.LastOperation(context.TODO(), instID, domain.PollDetails{OperationData: operationID})
		assert.NoError(t, err)

		// then
		assert.Equal(t, domain.LastOperation{
			State:       domain.Succeeded,
			Description: operationDescription,
		}, response)
	})
	t.Run("Should return last operation when operation ID not provided", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Operations().InsertProvisioningOperation(fixOperation())
		assert.NoError(t, err)

		err = memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: instID,
		})
		assert.NoError(t, err)

		lastOperationEndpoint := broker.NewLastOperation(memoryStorage.Operations(), memoryStorage.Instances(), logrus.StandardLogger())

		// when
		response, err := lastOperationEndpoint.LastOperation(context.TODO(), instID, domain.PollDetails{OperationData: ""})
		assert.NoError(t, err)

		// then
		assert.Equal(t, domain.LastOperation{
			State:       domain.Succeeded,
			Description: operationDescription,
		}, response)
	})
}

func fixOperation() internal.ProvisioningOperation {
	return internal.ProvisioningOperation{
		Operation: internal.Operation{
			ID:          operationID,
			InstanceID:  instID,
			State:       domain.Succeeded,
			Description: operationDescription,
		},
	}
}
