package azure

import (
	"context"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/azure"
)

type EventHub struct {
	HyperscalerProvider azure.HyperscalerProvider
	AccountProvider     hyperscaler.AccountProvider
	Context             context.Context
}
