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
	UpgradeShoot(accountID, runtimeID string, config schema.UpgradeShootInput) (schema.OperationStatus, error)
	ReconnectRuntimeAgent(accountID, runtimeID string) (string, error)
	RuntimeOperationStatus(accountID, operationID string) (schema.OperationStatus, error)
	RuntimeStatus(accountID, runtimeID string) (schema.RuntimeStatus, error)
}

type client struct {
	graphQLClient *gcli.Client
	queryProvider queryProvider
	graphqlizer   Graphqlizer
}

func NewProvisionerClient(endpoint string, queryDumping bool) Client {
	graphQlClient := gcli.NewClient(endpoint, gcli.WithHTTPClient(httputil.NewClient(120, false)))
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
		return schema.OperationStatus{}, errors.Wrap(err, "failed to provision a Runtime")
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
		return "", errors.Wrap(err, "Failed to deprovision Runtime")
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
		return schema.OperationStatus{}, errors.Wrap(err, "Failed to upgrade Runtime")
	}
	return res, nil
}

func (c *client) UpgradeShoot(accountID, runtimeID string, config schema.UpgradeShootInput) (schema.OperationStatus, error) {
	upgradeShootIptGQL, err := c.graphqlizer.UpgradeShootInputToGraphQL(config)
	if err != nil {
		return schema.OperationStatus{}, errors.Wrap(err, "Failed to convert Upgrade Shoot Input to query")
	}

	query := c.queryProvider.upgradeShoot(runtimeID, upgradeShootIptGQL)
	req := gcli.NewRequest(query)
	req.Header.Add(accountIDKey, accountID)

	var res schema.OperationStatus
	err = c.executeRequest(req, &res)
	if err != nil {
		return schema.OperationStatus{}, errors.Wrap(err, "Failed to upgrade Shoot")
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

func (c *client) RuntimeStatus(accountID, runtimeID string) (schema.RuntimeStatus, error) {
	query := c.queryProvider.runtimeStatus(runtimeID)
	req := gcli.NewRequest(query)
	req.Header.Add(accountIDKey, accountID)

	var response schema.RuntimeStatus
	err := c.executeRequest(req, &response)
	if err != nil {
		return schema.RuntimeStatus{}, errors.Wrap(err, "Failed to get Runtime status")
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
	switch {
	case isNotFoundError(err):
		return kebError.NotFoundError{}
	case isClientError(err):
		return err
	case err != nil:
		return kebError.WrapAsTemporaryError(err, "failed to execute the request")
	}

	return nil
}

func isClientError(err error) bool {
	if ee, ok := err.(gcli.ExtendedError); ok {
		code, found := ee.Extensions()["error_code"]
		if found {
			errCode := code.(float64)
			if errCode >= 400 && errCode < 500 {
				return true
			}
		}
	}
	return false
}

func isNotFoundError(err error) bool {
	if ee, ok := err.(gcli.ExtendedError); ok {
		reason, found := ee.Extensions()["error_reason"]
		if found {
			if reason == "err_db_not_found" {
				return true
			}
		}
	}
	return false
}

func OperationStatusLastError(lastErr *schema.LastError) kebError.ErrorReporter {
	var err kebError.LastError

	if lastErr == nil {
		return err.SetReason(kebError.ErrProvisionerNilLastError).SetComponent(kebError.ErrProvisioner)
	}

	return err.SetMessage(lastErr.ErrMessage).SetReason(kebError.ErrReason(lastErr.Reason)).SetComponent(kebError.ErrComponent(lastErr.Component))
}
