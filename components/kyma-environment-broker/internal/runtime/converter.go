package runtime

import (
	"strings"

	pkg "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type Converter interface {
	NewDTO(instance internal.Instance) (pkg.RuntimeDTO, error)
	ApplyProvisioningOperation(dto *pkg.RuntimeDTO, pOpr *internal.ProvisioningOperation)
	ApplyDeprovisioningOperation(dto *pkg.RuntimeDTO, dOpr *internal.DeprovisioningOperation)
	ApplyUpgradingKymaOperations(dto *pkg.RuntimeDTO, oprs []internal.UpgradeKymaOperation, totalCount int)
	ApplyUpgradingClusterOperations(dto *pkg.RuntimeDTO, oprs []internal.UpgradeClusterOperation, totalCount int)
	ApplyUpdateOperations(dto *pkg.RuntimeDTO, oprs []internal.UpdatingOperation, totalCount int)
	ApplySuspensionOperations(dto *pkg.RuntimeDTO, oprs []internal.DeprovisioningOperation)
	ApplyUnsuspensionOperations(dto *pkg.RuntimeDTO, oprs []internal.ProvisioningOperation)
}

type converter struct {
	defaultSubaccountRegion string
}

func NewConverter(platformRegion string) Converter {
	return &converter{
		defaultSubaccountRegion: platformRegion,
	}
}

func (c *converter) setRegionOrDefault(instance internal.Instance, runtime *pkg.RuntimeDTO) {
	if instance.Parameters.PlatformRegion == "" {
		runtime.SubAccountRegion = c.defaultSubaccountRegion
	} else {
		runtime.SubAccountRegion = instance.Parameters.PlatformRegion
	}
}

func (c *converter) ApplyProvisioningOperation(dto *pkg.RuntimeDTO, pOpr *internal.ProvisioningOperation) {
	if pOpr != nil {
		dto.Status.Provisioning = &pkg.Operation{}
		c.applyOperation(&pOpr.Operation, dto.Status.Provisioning)
	}
}

func (c *converter) ApplyDeprovisioningOperation(dto *pkg.RuntimeDTO, dOpr *internal.DeprovisioningOperation) {
	if dOpr != nil {
		dto.Status.Deprovisioning = &pkg.Operation{}
		c.applyOperation(&dOpr.Operation, dto.Status.Deprovisioning)
	}
}

func (c *converter) applyOperation(source *internal.Operation, target *pkg.Operation) {
	if source != nil {
		target.OperationID = source.ID
		target.CreatedAt = source.CreatedAt
		target.State = string(source.State)
		target.Description = source.Description
		target.OrchestrationID = source.OrchestrationID
	}
}

func (c *converter) NewDTO(instance internal.Instance) (pkg.RuntimeDTO, error) {
	toReturn := pkg.RuntimeDTO{
		InstanceID:                  instance.InstanceID,
		RuntimeID:                   instance.RuntimeID,
		GlobalAccountID:             instance.GlobalAccountID,
		SubscriptionGlobalAccountID: instance.SubscriptionGlobalAccountID,
		SubAccountID:                instance.SubAccountID,
		ServiceClassID:              instance.ServiceID,
		ServiceClassName:            instance.ServiceName,
		ServicePlanID:               instance.ServicePlanID,
		ServicePlanName:             instance.ServicePlanName,
		Provider:                    string(instance.Provider),
		ProviderRegion:              instance.ProviderRegion,
		UserID:                      instance.Parameters.ErsContext.UserID,
		Status: pkg.RuntimeStatus{
			CreatedAt:  instance.CreatedAt,
			ModifiedAt: instance.UpdatedAt,
		},
	}

	c.setRegionOrDefault(instance, &toReturn)

	urlSplitted := strings.Split(instance.DashboardURL, ".")
	if len(urlSplitted) > 1 {
		toReturn.ShootName = urlSplitted[1]
	}

	return toReturn, nil
}

func (c *converter) ApplyUpgradingKymaOperations(dto *pkg.RuntimeDTO, oprs []internal.UpgradeKymaOperation, totalCount int) {
	if len(oprs) <= 0 {
		return
	}
	dto.Status.UpgradingKyma = &pkg.OperationsData{}
	dto.Status.UpgradingKyma.TotalCount = totalCount
	dto.Status.UpgradingKyma.Count = len(oprs)
	dto.Status.UpgradingKyma.Data = make([]pkg.Operation, 0)
	for _, o := range oprs {
		op := pkg.Operation{}
		c.applyOperation(&o.Operation, &op)
		dto.Status.UpgradingKyma.Data = append(dto.Status.UpgradingKyma.Data, op)
	}
}

func (c *converter) ApplyUpgradingClusterOperations(dto *pkg.RuntimeDTO, oprs []internal.UpgradeClusterOperation, totalCount int) {
	if len(oprs) <= 0 {
		return
	}
	dto.Status.UpgradingCluster = &pkg.OperationsData{}
	dto.Status.UpgradingCluster.Data = make([]pkg.Operation, 0)
	for _, o := range oprs {
		op := pkg.Operation{}
		c.applyOperation(&o.Operation, &op)
		dto.Status.UpgradingCluster.Data = append(dto.Status.UpgradingCluster.Data, op)
	}
	dto.Status.UpgradingCluster.TotalCount = totalCount
	dto.Status.UpgradingCluster.Count = len(dto.Status.UpgradingCluster.Data)
}

func (c *converter) ApplySuspensionOperations(dto *pkg.RuntimeDTO, oprs []internal.DeprovisioningOperation) {
	if len(oprs) <= 0 {
		return
	}
	suspension := &pkg.OperationsData{}
	suspension.Data = make([]pkg.Operation, 0)

	for _, o := range oprs {
		if !o.Temporary {
			continue
		}
		op := pkg.Operation{}
		c.applyOperation(&o.Operation, &op)
		suspension.Data = append(suspension.Data, op)
	}
	suspension.TotalCount = len(suspension.Data)
	suspension.Count = len(suspension.Data)
	if suspension.Count > 0 {
		dto.Status.Suspension = suspension
	}
}

func (c *converter) ApplyUnsuspensionOperations(dto *pkg.RuntimeDTO, oprs []internal.ProvisioningOperation) {
	if len(oprs) <= 0 {
		return
	}
	dto.Status.Unsuspension = &pkg.OperationsData{}
	dto.Status.Unsuspension.Data = make([]pkg.Operation, 0)

	dto.Status.Unsuspension.TotalCount = len(oprs)
	dto.Status.Unsuspension.Count = len(oprs)

	for _, o := range oprs {
		op := pkg.Operation{}
		c.applyOperation(&o.Operation, &op)
		dto.Status.Unsuspension.Data = append(dto.Status.Unsuspension.Data, op)
	}
}

func (c *converter) ApplyUpdateOperations(dto *pkg.RuntimeDTO, oprs []internal.UpdatingOperation, totalCount int) {
	if len(oprs) <= 0 {
		return
	}

	dto.Status.Update = &pkg.OperationsData{}
	dto.Status.Update.Data = make([]pkg.Operation, 0)
	dto.Status.Update.Count = len(oprs)
	dto.Status.Update.TotalCount = totalCount
	for _, o := range oprs {
		op := pkg.Operation{}
		c.applyOperation(&o.Operation, &op)
		dto.Status.Update.Data = append(dto.Status.Update.Data, op)
	}
}
