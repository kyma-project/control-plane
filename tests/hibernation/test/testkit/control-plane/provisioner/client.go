package provisioner

import (
	"github.com/kyma-project/control-plane/tests/provisioner-tests/test/testkit/graphql"

	schema "github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	gcli "github.com/machinebox/graphql"
	"github.com/pkg/errors"
)

const (
	tenantHeader = "Tenant"
)

type Client interface {
	ProvisionRuntime(config schema.ProvisionRuntimeInput) (operationStatusID string, runtimeID string, err error)
	DeprovisionRuntime(runtimeID string) (string, error)
	RuntimeStatus(runtimeID string) (schema.RuntimeStatus, error)
	RuntimeOperationStatus(operationID string) (schema.OperationStatus, error)
	HibernateRuntime(runtimeID string) (string, error)
}

type client struct {
	tenant        string
	graphQLClient *graphql.Client
	queryProvider queryProvider
	graphqlizer   graphqlizer
}

func NewProvisionerClient(endpoint, tenant string, queryLogging bool) Client {
	return &client{
		tenant:        tenant,
		graphQLClient: graphql.NewGraphQLClient(endpoint, true, queryLogging),
		queryProvider: queryProvider{},
		graphqlizer:   graphqlizer{},
	}
}

func (c client) ProvisionRuntime(config schema.ProvisionRuntimeInput) (operationStatusID string, runtimeID string, err error) {
	provisionRuntimeIptGQL, err := c.graphqlizer.ProvisionRuntimeInputToGraphQL(config)
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to convert Provision Runtime Input to query")
	}

	query := c.queryProvider.provisionRuntime(provisionRuntimeIptGQL)
	req := c.newRequest(query)

	var operationStatus schema.OperationStatus
	err = c.graphQLClient.ExecuteRequest(req, &operationStatus)
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to provision Runtime")
	}
	if operationStatus.ID == nil || operationStatus.RuntimeID == nil {
		return "", "", errors.New("Failed to receive proper Operation Status response")
	}
	return *operationStatus.ID, *operationStatus.RuntimeID, nil
}

func (c client) DeprovisionRuntime(runtimeID string) (string, error) {
	query := c.queryProvider.deprovisionRuntime(runtimeID)
	req := c.newRequest(query)

	var operationId string
	err := c.graphQLClient.ExecuteRequest(req, &operationId)
	if err != nil {
		return "", errors.Wrap(err, "Failed to deprovision Runtime")
	}
	return operationId, nil
}

func (c client) RuntimeStatus(runtimeID string) (schema.RuntimeStatus, error) {
	query := c.queryProvider.runtimeStatus(runtimeID)
	req := c.newRequest(query)

	var response schema.RuntimeStatus
	err := c.graphQLClient.ExecuteRequest(req, &response)
	if err != nil {
		return schema.RuntimeStatus{}, errors.Wrap(err, "Failed to get Runtime status")
	}
	return response, nil
}

func (c client) RuntimeOperationStatus(operationID string) (schema.OperationStatus, error) {
	query := c.queryProvider.runtimeOperationStatus(operationID)
	req := c.newRequest(query)

	var response schema.OperationStatus
	err := c.graphQLClient.ExecuteRequest(req, &response)
	if err != nil {
		return schema.OperationStatus{}, errors.Wrap(err, "Failed to get Runtime operation status")
	}
	return response, nil
}

func (c client) HibernateRuntime(runtimeID string) (string, error) {
	query := c.queryProvider.hibernateCluster(runtimeID)
	req := c.newRequest(query)

	var operationStatus schema.OperationStatus
	err := c.graphQLClient.ExecuteRequest(req, &operationStatus)
	if err != nil {
		return "", errors.Wrap(err, "Failed to hibernate runtime")
	}
	if operationStatus.ID == nil || operationStatus.RuntimeID == nil {
		return "", errors.New("Failed to receive proper Operation Status response")
	}
	return *operationStatus.ID, nil
}

func (c client) newRequest(query string) *gcli.Request {
	req := gcli.NewRequest(query)

	req.Header.Add(tenantHeader, c.tenant)

	return req
}
