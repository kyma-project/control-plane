package director

import (
	"context"
	"testing"

	"github.com/kyma-incubator/compass/components/director/pkg/graphql"
	mocks "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/director/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	machineGraphql "github.com/machinebox/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestClient_SetLabel(t *testing.T) {
	// given
	var (
		accountID  = "ad568853-ecf3-433a-8638-e53aa6bead5d"
		runtimeID  = "775dc85e-825b-4ddf-abf6-da0dd002b66e"
		labelKey   = "testKey"
		labelValue = "testValue"
	)

	qc := &mocks.GraphQLClient{}
	cfg := Config{}

	client := NewDirectorClient(context.Background(), cfg, logger.NewLogDummy())
	client.graphQLClient = qc

	request := createGraphQLLabelRequest(client, accountID, runtimeID, labelKey, labelValue)

	qc.On("Run", context.Background(), request, mock.AnythingOfType("*director.runtimeLabelResponse")).Run(func(args mock.Arguments) {
		arg, ok := args.Get(2).(*runtimeLabelResponse)
		if !ok {
			return
		}
		arg.Result = &graphql.Label{
			Key:   labelKey,
			Value: labelValue,
		}
	}).Return(nil)
	defer qc.AssertExpectations(t)

	// when
	err := client.SetLabel(accountID, runtimeID, labelKey, labelValue)

	// then
	assert.NoError(t, err)
}

func createGraphQLRequest(client *Client, accountID, runtimeID string) *machineGraphql.Request {
	query := client.queryProvider.Runtime(runtimeID)
	request := machineGraphql.NewRequest(query)
	request.Header.Add(accountIDKey, accountID)

	return request
}

func createGraphQLLabelRequest(client *Client, accountID, runtimeID, key, label string) *machineGraphql.Request {
	query := client.queryProvider.SetRuntimeLabel(runtimeID, key, label)
	request := machineGraphql.NewRequest(query)
	request.Header.Add(accountIDKey, accountID)

	return request
}
