package director

import (
	"context"
	"fmt"
	"testing"

	"github.com/kyma-incubator/compass/components/director/pkg/graphql"
	mocks "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/director/automock"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	machineGraphql "github.com/machinebox/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestClient_GetConsoleURL(t *testing.T) {
	var (
		runtimeID   = "620f2303-f084-4956-8594-b351fbff124d"
		accountID   = "32f2e45c-74dc-4bb8-b03f-7cb6a44c1fd9"
		expectedURL = "http://example.com"
	)

	t.Run("url returned successfully", func(t *testing.T) {
		// Given
		qc := &mocks.GraphQLClient{}
		cfg := Config{}

		client := NewDirectorClient(context.Background(), cfg, logger.NewLogDummy())
		client.graphQLClient = qc

		// #create request
		request := createGraphQLRequest(client, accountID, runtimeID)

		// #mock on Run method for grapQL client
		qc.On("Run", context.Background(), request, mock.AnythingOfType("*director.getURLResponse")).Run(func(args mock.Arguments) {
			arg, ok := args.Get(2).(*getURLResponse)
			if !ok {
				return
			}
			arg.Result = graphql.RuntimeExt{
				Runtime: graphql.Runtime{
					Status: &graphql.RuntimeStatus{
						Condition: graphql.RuntimeStatusConditionConnected,
					},
				},
				Labels: map[string]interface{}{
					consoleURLLabelKey: expectedURL,
				},
			}
		}).Return(nil)
		defer qc.AssertExpectations(t)

		// When
		URL, tokenErr := client.GetConsoleURL(accountID, runtimeID)

		// Then
		assert.NoError(t, tokenErr)
		assert.False(t, kebError.IsTemporaryError(tokenErr))
		assert.Equal(t, expectedURL, URL)
	})

	t.Run("response from director is empty", func(t *testing.T) {
		// Given
		qc := &mocks.GraphQLClient{}

		client := NewDirectorClient(context.Background(), Config{}, logger.NewLogDummy())
		client.graphQLClient = qc

		// #create request
		request := createGraphQLRequest(client, accountID, runtimeID)

		// #mock on Run method for grapQL client
		qc.On("Run", context.Background(), request, mock.AnythingOfType("*director.getURLResponse")).Return(nil)
		defer qc.AssertExpectations(t)

		// When
		URL, tokenErr := client.GetConsoleURL(accountID, runtimeID)

		// Then
		assert.Error(t, tokenErr)
		assert.True(t, kebError.IsTemporaryError(tokenErr))
		assert.Equal(t, "", URL)
	})

	t.Run("response from director is in failed state", func(t *testing.T) {
		// Given
		qc := &mocks.GraphQLClient{}

		client := NewDirectorClient(context.Background(), Config{}, logger.NewLogDummy())
		client.graphQLClient = qc

		// #create request
		request := createGraphQLRequest(client, accountID, runtimeID)

		// #mock on Run method for grapQL client
		qc.On("Run", context.Background(), request, mock.AnythingOfType("*director.getURLResponse")).Run(func(args mock.Arguments) {
			arg, ok := args.Get(2).(*getURLResponse)
			if !ok {
				return
			}
			arg.Result = graphql.RuntimeExt{
				Runtime: graphql.Runtime{
					Status: &graphql.RuntimeStatus{
						Condition: graphql.RuntimeStatusConditionFailed,
					},
				},
				Labels: map[string]interface{}{
					consoleURLLabelKey: "",
				},
			}
		}).Return(nil)
		defer qc.AssertExpectations(t)

		// When
		URL, tokenErr := client.GetConsoleURL(accountID, runtimeID)

		// Then
		assert.Error(t, tokenErr)
		assert.False(t, kebError.IsTemporaryError(tokenErr))
		assert.Equal(t, "", URL)
	})

	t.Run("response from director has no proper labels", func(t *testing.T) {
		// Given
		qc := &mocks.GraphQLClient{}

		client := NewDirectorClient(context.Background(), Config{}, logger.NewLogDummy())
		client.graphQLClient = qc

		// #create request
		request := createGraphQLRequest(client, accountID, runtimeID)

		// #mock on Run method for grapQL client
		qc.On("Run", context.Background(), request, mock.AnythingOfType("*director.getURLResponse")).Run(func(args mock.Arguments) {
			arg, ok := args.Get(2).(*getURLResponse)
			if !ok {
				return
			}
			arg.Result = graphql.RuntimeExt{
				Runtime: graphql.Runtime{
					Status: &graphql.RuntimeStatus{
						Condition: graphql.RuntimeStatusConditionConnected,
					},
				},
				Labels: map[string]interface{}{
					"wrongURLLabel": expectedURL,
				},
			}
		}).Return(nil)
		defer qc.AssertExpectations(t)

		// When
		URL, tokenErr := client.GetConsoleURL(accountID, runtimeID)

		// Then
		assert.Error(t, tokenErr)
		assert.True(t, kebError.IsTemporaryError(tokenErr))
		assert.Equal(t, "", URL)
	})

	t.Run("response from director has label with wrong type", func(t *testing.T) {
		// Given
		qc := &mocks.GraphQLClient{}

		client := NewDirectorClient(context.Background(), Config{}, logger.NewLogDummy())
		client.graphQLClient = qc

		// #create request
		request := createGraphQLRequest(client, accountID, runtimeID)

		// #mock on Run method for grapQL client
		qc.On("Run", context.Background(), request, mock.AnythingOfType("*director.getURLResponse")).Run(func(args mock.Arguments) {
			arg, ok := args.Get(2).(*getURLResponse)
			if !ok {
				return
			}
			arg.Result = graphql.RuntimeExt{
				Runtime: graphql.Runtime{
					Status: &graphql.RuntimeStatus{
						Condition: graphql.RuntimeStatusConditionConnected,
					},
				},
				Labels: map[string]interface{}{
					consoleURLLabelKey: 42,
				},
			}
		}).Return(nil)
		defer qc.AssertExpectations(t)

		// When
		URL, tokenErr := client.GetConsoleURL(accountID, runtimeID)

		// Then
		assert.Error(t, tokenErr)
		assert.False(t, kebError.IsTemporaryError(tokenErr))
		assert.Equal(t, "", URL)
	})

	t.Run("response from director has wrong URL value", func(t *testing.T) {
		// Given
		qc := &mocks.GraphQLClient{}

		client := NewDirectorClient(context.Background(), Config{}, logger.NewLogDummy())
		client.graphQLClient = qc

		// #create request
		request := createGraphQLRequest(client, accountID, runtimeID)

		// #mock on Run method for grapQL client
		qc.On("Run", context.Background(), request, mock.AnythingOfType("*director.getURLResponse")).Run(func(args mock.Arguments) {
			arg, ok := args.Get(2).(*getURLResponse)
			if !ok {
				return
			}
			arg.Result = graphql.RuntimeExt{
				Runtime: graphql.Runtime{
					Status: &graphql.RuntimeStatus{
						Condition: graphql.RuntimeStatusConditionConnected,
					},
				},
				Labels: map[string]interface{}{
					consoleURLLabelKey: "wrong-URL",
				},
			}
		}).Return(nil)
		defer qc.AssertExpectations(t)

		// When
		URL, tokenErr := client.GetConsoleURL(accountID, runtimeID)

		// Then
		assert.Error(t, tokenErr)
		assert.False(t, kebError.IsTemporaryError(tokenErr))
		assert.Equal(t, "", URL)
	})

	t.Run("client graphQL returns error", func(t *testing.T) {
		// Given
		qc := &mocks.GraphQLClient{}
		cfg := Config{}

		client := NewDirectorClient(context.Background(), cfg, logger.NewLogDummy())
		client.graphQLClient = qc

		// #create request
		request := createGraphQLRequest(client, accountID, runtimeID)

		// #mock on Run method for grapQL client
		qc.On("Run", context.Background(), request, mock.AnythingOfType("*director.getURLResponse")).Return(fmt.Errorf("director error"))
		defer qc.AssertExpectations(t)

		// When
		URL, tokenErr := client.GetConsoleURL(accountID, runtimeID)

		// Then
		assert.Error(t, tokenErr)
		assert.True(t, kebError.IsTemporaryError(tokenErr))
		assert.Equal(t, "", URL)
	})
}

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
