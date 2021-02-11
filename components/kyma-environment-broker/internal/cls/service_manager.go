package cls

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
)

var (
	regionMap = map[string]string{
		"westeurope":    "eu",
		"northeurope":   "eu",
		"westus2":       "eu",
		"uksouth":       "eu",
		"francecentral": "eu",
		"uaenorth":      "eu",

		"eastus":      "us",
		"eastus2":     "us",
		"centralus":   "us",
		"eastus2euap": "us",
	}
)

const (
	defaultServiceManagerRegion = "eu"
)

//ServiceManagerClient creates an instance of servicemanager.Client that is initialized with credentials for the current SKR region
func ServiceManagerClient(factory internal.SMClientFactory, config *ServiceManagerConfig, skrRegion *string) (servicemanager.Client, error) {
	serviceManagerRegion, err := determineServiceManagerRegion(skrRegion)
	if err != nil {
		return nil, err
	}

	credentials := findCredentials(config, serviceManagerRegion)
	if credentials == nil {
		return nil, fmt.Errorf("unable find credentials for the region: %s", serviceManagerRegion)
	}

	return factory.ForCredentials(credentials), nil
}

func determineServiceManagerRegion(skrRegion *string) (string, error) {
	if skrRegion == nil {
		return defaultServiceManagerRegion, nil
	}

	serviceManagerRegion, exists := regionMap[*skrRegion]
	if !exists {
		return "", fmt.Errorf("unsupported region: %s", *skrRegion)
	}

	return serviceManagerRegion, nil
}

func findCredentials(config *ServiceManagerConfig, region string) *servicemanager.Credentials {
	for _, credentials := range config.Credentials {
		if string(credentials.Region) == region {
			return &servicemanager.Credentials{
				URL:      credentials.URL,
				Username: credentials.Username,
				Password: credentials.Password,
			}
		}
	}

	return nil
}
