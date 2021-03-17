package cls

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/sirupsen/logrus"
)

var (
	azureRegionsForClsEu = []string{
		"northeurope",
		"westeurope",
		"uksouth",
		"ukwest",
		"southafricanorth",
		"southafricawest",
		"francecentral",
		"francesouth",
		"germanywestcentral",
		"germanynorth",
		"norwayeast",
		"norwaywest",
		"switzerlandnorth",
		"switzerlandwest",
		"uaenorth",
		"uaecentral",
		"europe",
		"uk",
	}

	azureRegionsForClsUs = []string{
		"eastus",
		"eastus2",
		"southcentralus",
		"westus2",
		"australiaeast",
		"southeastasia",
		"centralus",
		"northcentralus",
		"westus",
		"centralindia",
		"eastasia",
		"japaneast",
		"koreacentral",
		"canadacentral",
		"brazilsouth",
		"centralusstage",
		"eastusstage",
		"eastus2stage",
		"northcentralusstage",
		"southcentralusstage",
		"westusstage",
		"westus2stage",
		"asia",
		"asiapacific",
		"australia",
		"brazil",
		"canada",
		"india",
		"japan",
		"unitedstates",
		"eastasiastage",
		"southeastasiastage",
		"centraluseuap",
		"eastus2euap",
		"westcentralus",
		"westus3",
		"australiacentral",
		"australiacentral2",
		"australiasoutheast",
		"japanwest",
		"koreasouth",
		"southindia",
		"westindia",
		"canadaeast",
		"brazilsoutheast",
	}
)

const (
	fallbackServiceManagerRegion = RegionEurope
)

//DetermineServiceManagerRegion maps a hyperscaler-specific region (currently, Azure only) to a region where a CLS instance is to be provisioned. Returns eu as a fallback regions.
func DetermineServiceManagerRegion(skrRegion *string, log logrus.FieldLogger) string {
	if skrRegion == nil {
		log.Warnf("No region provided, falling back to %s", fallbackServiceManagerRegion)
		return fallbackServiceManagerRegion
	}

	for _, region := range azureRegionsForClsEu {
		if region == *skrRegion {
			return RegionEurope
		}
	}

	for _, region := range azureRegionsForClsUs {
		if region == *skrRegion {
			return RegionUS
		}
	}

	log.Warnf("Unknown region %s, falling back to %s", *skrRegion, fallbackServiceManagerRegion)

	return fallbackServiceManagerRegion
}

//FindCredentials searches for Service Manager credentials for a given region
func FindCredentials(config *ServiceManagerConfig, region string) (*servicemanager.Credentials, error) {
	for _, credentials := range config.Credentials {
		if credentials.Region == region {
			return &servicemanager.Credentials{
				URL:      credentials.URL,
				Username: credentials.Username,
				Password: credentials.Password,
			}, nil
		}
	}

	return nil, fmt.Errorf("unable to find credentials for region: %s", region)
}
