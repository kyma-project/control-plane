package cloudprovider

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type azureResourceCleaner struct {
	azureClient resources.GroupsClient
}

type config struct {
	clientID       string
	clientSecret   string
	subscriptionID string
	tenantID       string
	userAgent      string
}

func NewAzureResourcesCleaner(secretData map[string][]byte) (ResourceCleaner, error) {
	config, err := toConfig(secretData)
	if err != nil {
		return nil, err
	}

	azureClient, err := newResourceGroupsClient(config)
	if err != nil {
		return nil, err
	}

	return &azureResourceCleaner{
		azureClient: azureClient,
	}, nil
}

func (ac azureResourceCleaner) Do() error {
	ctx := context.Background()
	resourceGroups, err := ac.azureClient.List(ctx, "", nil)
	if err != nil {
		return err
	}

	for _, resourceGroup := range resourceGroups.Values() {
		if resourceGroup.Name != nil {
			log.Infof("Deleting resource group '%s'", *resourceGroup.Name)
			future, err := ac.azureClient.Delete(ctx, *resourceGroup.Name)
			if err != nil {
				log.Errorf("failed to init resource group '%s' deletion", *resourceGroup.Name)
				continue
			}

			err = future.WaitForCompletionRef(ctx, ac.azureClient.Client)
			if err != nil {
				log.Errorf("failed to remove resource group '%s', %s: ", *resourceGroup.Name, err.Error())
			}
		}
	}

	return nil
}

func toConfig(secretData map[string][]byte) (config, error) {
	clientID, exists := secretData["clientID"]
	if !exists {
		return config{}, errors.New("clientID not provided in the secret")
	}

	clientSecret, exists := secretData["clientSecret"]
	if !exists {
		return config{}, errors.New("clientSecret not provided in the secret")
	}

	subscriptionID, exists := secretData["subscriptionID"]
	if !exists {
		return config{}, errors.New("subscriptionID not provided in the secret")
	}

	tenantID, exists := secretData["tenantID"]
	if !exists {
		return config{}, errors.New("tenantID not provided in the secret")
	}

	return config{
		clientID:       string(clientID),
		clientSecret:   string(clientSecret),
		subscriptionID: string(subscriptionID),
		tenantID:       string(tenantID),
		userAgent:      "kyma-environment-broker",
	}, nil
}

func newResourceGroupsClient(config config) (resources.GroupsClient, error) {
	azureEnv, err := azure.EnvironmentFromName("AzurePublicCloud") // shouldn't fail
	if err != nil {
		return resources.GroupsClient{}, err
	}

	authorizer, err := getResourceManagementAuthorizer(&config, &azureEnv)
	if err != nil {
		return resources.GroupsClient{}, err
	}

	return getGroupsClient(&config, authorizer)
}

// getGroupsClient gets a client for handling of Azure ResourceGroups
func getGroupsClient(config *config, authorizer autorest.Authorizer) (resources.GroupsClient, error) {
	client := resources.NewGroupsClient(config.subscriptionID)
	client.Authorizer = authorizer

	if err := client.AddToUserAgent(config.userAgent); err != nil {
		return resources.GroupsClient{}, errors.Wrapf(err, "while adding user agent [%s]", config.userAgent)
	}

	return client, nil
}

func getResourceManagementAuthorizer(config *config, environment *azure.Environment) (autorest.Authorizer, error) {
	armAuthorizer, err := getAuthorizerForResource(config, environment)
	if err != nil {
		return nil, errors.Wrap(err, "while creating resource authorizer")
	}

	return armAuthorizer, err
}

func getAuthorizerForResource(config *config, environment *azure.Environment) (autorest.Authorizer, error) {
	oauthConfig, err := adal.NewOAuthConfig(environment.ActiveDirectoryEndpoint, config.tenantID)
	if err != nil {
		return nil, err
	}

	token, err := adal.NewServicePrincipalToken(*oauthConfig, config.clientID, config.clientSecret, environment.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}
	return autorest.NewBearerAuthorizer(token), err
}
