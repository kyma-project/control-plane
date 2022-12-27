package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/eventhub/mgmt/2017-04-01/eventhub"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
)

type HyperscalerProvider interface {
	GetClient(config *Config, logger logrus.FieldLogger) (Interface, error)
}

var _ HyperscalerProvider = (*azureProvider)(nil)

type azureProvider struct{}

func NewAzureProvider() HyperscalerProvider {
	return &azureProvider{}
}

// GetClient gets a client for interacting with Azure
func (ac *azureProvider) GetClient(config *Config, logger logrus.FieldLogger) (Interface, error) {

	environment, err := config.Environment()
	if err != nil {
		return nil, err
	}

	authorizer, err := ac.getResourceManagementAuthorizer(config, environment)
	if err != nil {
		return nil, fmt.Errorf("while initializing authorizer: %w", err)
	}

	// create namespace client
	nsClient, err := ac.getNamespaceClient(config, authorizer)
	if err != nil {
		return nil, fmt.Errorf("while creating namespace client: %w", err)
	}

	// create resource group client
	resourceGroupClient, err := ac.getGroupsClient(config, authorizer)
	if err != nil {
		return nil, fmt.Errorf("while creating resource group client: %w", err)
	}

	// create azure client
	return NewAzureClient(nsClient, resourceGroupClient, logger), nil
}

// getGroupsClient gets a client for handling of Azure Namespaces
func (ac *azureProvider) getNamespaceClient(config *Config, authorizer autorest.Authorizer) (eventhub.NamespacesClient, error) {
	nsClient := eventhub.NewNamespacesClient(config.subscriptionID)
	nsClient.Authorizer = authorizer

	if err := nsClient.AddToUserAgent(config.userAgent); err != nil {
		return eventhub.NamespacesClient{}, fmt.Errorf("while adding user agent [%s]: %w", config.userAgent, err)
	}
	return nsClient, nil
}

// getGroupsClient gets a client for handling of Azure ResourceGroups
func (ac *azureProvider) getGroupsClient(config *Config, authorizer autorest.Authorizer) (resources.GroupsClient, error) {
	client := resources.NewGroupsClient(config.subscriptionID)
	client.Authorizer = authorizer

	if err := client.AddToUserAgent(config.userAgent); err != nil {
		return resources.GroupsClient{}, fmt.Errorf("while adding user agent [%s]: %w", config.userAgent, err)
	}

	return client, nil
}

func (ac *azureProvider) getResourceManagementAuthorizer(config *Config, environment *azure.Environment) (autorest.Authorizer, error) {
	armAuthorizer, err := ac.getAuthorizerForResource(config, environment)
	if err != nil {
		return nil, fmt.Errorf("while creating resource authorizer: %w", err)
	}

	return armAuthorizer, err
}

func (ac *azureProvider) getAuthorizerForResource(config *Config, environment *azure.Environment) (autorest.Authorizer, error) {

	oauthConfig, err := adal.NewOAuthConfig(environment.ActiveDirectoryEndpoint, config.tenantID)
	if err != nil {
		return nil, fmt.Errorf("while creating OAuth config: %w", err)
	}

	token, err := adal.NewServicePrincipalToken(*oauthConfig, config.clientID, config.clientSecret, environment.ResourceManagerEndpoint)
	if err != nil {
		return nil, fmt.Errorf("while creating service principal token: %w", err)
	}
	return autorest.NewBearerAuthorizer(token), nil
}
