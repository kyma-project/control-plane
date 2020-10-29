package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/azure"
)

const (
	// prefix is added before the created Azure resources
	// to satisfy Azure naming validation: https://docs.microsoft.com/en-us/rest/api/servicebus/create-namespace
	prefix = "k"
)

type ProviderContext struct {
	HyperscalerProvider azure.HyperscalerProvider
	AccountProvider     hyperscaler.AccountProvider
	Context             context.Context
}

// getAzureResourceName returns a valid Azure resource name that is in lower case and starts with a letter.
func GetAzureResourceName(name string) string {
	name = fmt.Sprintf("%s%s", prefix, name)
	name = strings.ToLower(name)
	return name
}
