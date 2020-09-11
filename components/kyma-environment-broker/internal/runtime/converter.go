package runtime

import (
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type converter struct {
}

func NewConverter() *converter {
	return &converter{}
}

func (c *converter) InstancesAndOperationsToDTO(instance internal.Instance, pOpr *internal.ProvisioningOperation,
	dOpr *internal.DeprovisioningOperation, ukOpr *internal.UpgradeKymaOperation) runtimeDTO {
	toReturn := runtimeDTO{
		InstanceID:      instance.InstanceID,
		RuntimeID:       instance.RuntimeID,
		GlobalAccountID: instance.GlobalAccountID,
		SubAccountID:    instance.SubAccountID,
	}
	urlSplitted := strings.Split(instance.DashboardURL, ".")
	if len(urlSplitted) > 1 {
		toReturn.ShootName = urlSplitted[1]
	}
	if pOpr != nil {
		toReturn.ProvisioningState = string(pOpr.State)
	}

	if dOpr != nil {
		toReturn.DeprovisioningState = string(dOpr.State)
	}

	if ukOpr != nil {
		toReturn.UpgradeState = string(ukOpr.State)
	}
	return toReturn
}
