package cls

import (
	"fmt"

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

//DetermineServiceManagerRegion maps a hyperscaler-specific region (currently, Azure only) to a region where a CLS instance is to be provisioned
func DetermineServiceManagerRegion(skrRegion *string) (string, error) {
	if skrRegion == nil {
		return defaultServiceManagerRegion, nil
	}

	serviceManagerRegion, exists := regionMap[*skrRegion]
	if !exists {
		return "", fmt.Errorf("unsupported region: %s", *skrRegion)
	}

	return serviceManagerRegion, nil
}

//FindCredentials searches for Service Manager credentials for a given region
func FindCredentials(config *ServiceManagerConfig, region string) (*servicemanager.Credentials, error) {
	for _, credentials := range config.Credentials {
		if string(credentials.Region) == region {
			return &servicemanager.Credentials{
				URL:      credentials.URL,
				Username: credentials.Username,
				Password: credentials.Password,
			}, nil
		}
	}

	return nil, fmt.Errorf("unable to find credentials for region: %s", region)
}
