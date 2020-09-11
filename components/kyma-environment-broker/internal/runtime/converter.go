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
		InstanceID:       instance.InstanceID,
		RuntimeID:        instance.RuntimeID,
		GlobalAccountID:  instance.GlobalAccountID,
		SubAccountID:     instance.SubAccountID,
		ServiceClassID:   instance.ServiceID,
		ServiceClassName: instance.ServiceName,
		ServicePlanID:    instance.ServicePlanID,
		ServicePlanName:  instance.ServicePlanName,
		Status: runtimeStatus{
			CreatedAt: instance.CreatedAt,
		},
	}

	urlSplitted := strings.Split(instance.DashboardURL, ".")
	if len(urlSplitted) > 1 {
		toReturn.ShootName = urlSplitted[1]
	}
	if pOpr != nil {
		toReturn.Status.Provisioning.State = string(pOpr.State)
		toReturn.Status.Provisioning.Description = pOpr.Description
	}

	if dOpr != nil {
		toReturn.Status.DeletedAt = &instance.DeletedAt
		toReturn.Status.Deprovisioning.State = string(dOpr.State)
		toReturn.Status.Deprovisioning.Description = dOpr.Description
	}

	if ukOpr != nil {
		toReturn.Status.UpdatedAt = &instance.UpdatedAt
		toReturn.Status.UpgradingKyma.State = string(ukOpr.State)
		toReturn.Status.UpgradingKyma.Description = ukOpr.Description
	}
	return toReturn
}
