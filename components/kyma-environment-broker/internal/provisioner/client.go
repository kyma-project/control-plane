package provisioner

import (
	"context"
	"fmt"
	"reflect"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"

	gcli "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/third_party/machinebox/graphql"
	schema "github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
)

// accountIDKey is a header key name for request send by graphQL client
const (
	accountIDKey    = "tenant"
	subAccountIDKey = "sub-account"
)

//go:generate mockery -name=Client -output=automock -outpkg=automock -case=underscore

type Client interface {
	ProvisionRuntime(accountID, subAccountID string, config schema.ProvisionRuntimeInput) (schema.OperationStatus, error)
	DeprovisionRuntime(accountID, runtimeID string) (string, error)
	UpgradeRuntime(accountID, runtimeID string, config schema.UpgradeRuntimeInput) (schema.OperationStatus, error)
	ReconnectRuntimeAgent(accountID, runtimeID string) (string, error)
	RuntimeOperationStatus(accountID, operationID string) (schema.OperationStatus, error)
}

type client struct {
	graphQLClient *gcli.Client
	queryProvider queryProvider
	graphqlizer   Graphqlizer
}

func NewProvisionerClient(endpoint string, queryDumping bool) Client {
	graphQlClient := gcli.NewClient(endpoint, gcli.WithHTTPClient(httputil.NewClient(30, false)))
	if queryDumping {
		graphQlClient.Log = func(s string) {
			fmt.Println(s)
		}
	}

	return &client{
		graphQLClient: graphQlClient,
		queryProvider: queryProvider{},
		graphqlizer:   Graphqlizer{},
	}
}

func (c *client) ProvisionRuntime(accountID, subAccountID string, config schema.ProvisionRuntimeInput) (schema.OperationStatus, error) {
	provisionRuntimeIptGQL, err := c.graphqlizer.ProvisionRuntimeInputToGraphQL(config)
	if err != nil {
		return schema.OperationStatus{}, errors.Wrap(err, "Failed to convert Provision Runtime Input to query")
	}

	query := c.queryProvider.provisionRuntime(provisionRuntimeIptGQL)
	req := gcli.NewRequest(query)
	req.Header.Add(accountIDKey, accountID)
	req.Header.Add(subAccountIDKey, subAccountID)

	var response schema.OperationStatus
	err = c.executeRequest(req, &response)
	if err != nil {
		return schema.OperationStatus{}, errors.Wrapf(err, "failed to provision a Runtime")
	}

	return response, nil
}

func (c *client) DeprovisionRuntime(accountID, runtimeID string) (string, error) {
	query := c.queryProvider.deprovisionRuntime(runtimeID)
	req := gcli.NewRequest(query)
	req.Header.Add(accountIDKey, accountID)

	var operationId string
	err := c.executeRequest(req, &operationId)
	if err != nil {
		return "", fmt.Errorf("Failed to deprovision Runtime: %w", err)
	}
	return operationId, nil
}

func (c *client) UpgradeRuntime(accountID, runtimeID string, config schema.UpgradeRuntimeInput) (schema.OperationStatus, error) {
	upgradeRuntimeIptGQL, err := c.graphqlizer.UpgradeRuntimeInputToGraphQL(config)
	if err != nil {
		return schema.OperationStatus{}, errors.Wrap(err, "Failed to convert Upgrade Runtime Input to query")
	}

	query := c.queryProvider.upgradeRuntime(runtimeID, upgradeRuntimeIptGQL)
	req := gcli.NewRequest(query)
	req.Header.Add(accountIDKey, accountID)

	var res schema.OperationStatus
	err = c.executeRequest(req, &res)
	if err != nil {
		return schema.OperationStatus{}, fmt.Errorf("Failed to upgrade Runtime: %w", err)
	}
	return res, nil
}

func (c *client) ReconnectRuntimeAgent(accountID, runtimeID string) (string, error) {
	query := c.queryProvider.reconnectRuntimeAgent(runtimeID)
	req := gcli.NewRequest(query)
	req.Header.Add(accountIDKey, accountID)

	var operationId string
	err := c.executeRequest(req, &operationId)
	if err != nil {
		return "", errors.Wrap(err, "Failed to reconnect Runtime agent")
	}
	return operationId, nil
}

func (c *client) RuntimeOperationStatus(accountID, operationID string) (schema.OperationStatus, error) {
	query := c.queryProvider.runtimeOperationStatus(operationID)
	req := gcli.NewRequest(query)
	req.Header.Add(accountIDKey, accountID)

	var response schema.OperationStatus
	err := c.executeRequest(req, &response)
	if err != nil {
		return schema.OperationStatus{}, errors.Wrap(err, "Failed to get Runtime operation status")
	}
	return response, nil
}

func (c *client) executeRequest(req *gcli.Request, respDestination interface{}) error {
	if reflect.ValueOf(respDestination).Kind() != reflect.Ptr {
		return errors.New("destination is not of pointer type")
	}

	type graphQLResponseWrapper struct {
		Result interface{} `json:"result"`
	}

	wrapper := &graphQLResponseWrapper{Result: respDestination}
	err := c.graphQLClient.Run(context.TODO(), req, wrapper)
	if ee, ok := err.(gcli.ExtendedError); ok {
		code, found := ee.Extensions()["error_code"]
		if found {
			errCode := code.(float64)
			if errCode >= 400 && errCode < 500 {
				return err
			}
		}
	}
	if err != nil {
		return kebError.AsTemporaryError(err, "failed to execute the request")
	}

	return nil
}

type BadRequestError struct {
	err error
}

func (e BadRequestError) Error() string {
	return fmt.Sprintf("bad request error: %s", e.err.Error())
}
