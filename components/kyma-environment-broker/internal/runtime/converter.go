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
		InstanceID:       instance.InstanceID,
		RuntimeID:        instance.RuntimeID,
		GlobalAccountID:  instance.GlobalAccountID,
		SubAccountID:     instance.SubAccountID,
		ServiceClassID:   instance.ServiceID,
		ServiceClassName: instance.ServiceName,
		ServicePlanID:    instance.ServicePlanID,
		ServicePlanName:  instance.ServicePlanName,
		ProviderRegion:   instance.ProviderRegion,
		Status: pkg.RuntimeStatus{
			CreatedAt:    instance.CreatedAt,
			ModifiedAt:   instance.UpdatedAt,
			Provisioning: &pkg.Operation{},
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
	dto.Status.UpgradingKyma.TotalCount = totalCount
	dto.Status.UpgradingKyma.Count = len(oprs)
	dto.Status.UpgradingKyma.Data = make([]pkg.Operation, 0)
	for _, o := range oprs {
		op := pkg.Operation{}
		c.applyOperation(&o.Operation, &op)
		dto.Status.UpgradingKyma.Data = append(dto.Status.UpgradingKyma.Data, op)
	}
}

func (c *converter) ApplySuspensionOperations(dto *pkg.RuntimeDTO, oprs []internal.DeprovisioningOperation) {
	dto.Status.Suspension.Data = make([]pkg.Operation, 0)

	for _, o := range oprs {
		if !o.Temporary {
			continue
		}
		op := pkg.Operation{}
		c.applyOperation(&o.Operation, &op)
		dto.Status.Suspension.Data = append(dto.Status.Suspension.Data, op)
	}
	dto.Status.Suspension.TotalCount = len(dto.Status.Suspension.Data)
	dto.Status.Suspension.Count = len(dto.Status.Suspension.Data)
}

func (c *converter) ApplyUnsuspensionOperations(dto *pkg.RuntimeDTO, oprs []internal.ProvisioningOperation) {
	dto.Status.Unsuspension.Data = make([]pkg.Operation, 0)
	if len(oprs) <= 1 {
		return
	}

	unsuspensionOps := oprs[:len(oprs)-1]

	dto.Status.Unsuspension.TotalCount = len(unsuspensionOps)
	dto.Status.Unsuspension.Count = len(unsuspensionOps)

	for _, o := range unsuspensionOps {
		op := pkg.Operation{}
		c.applyOperation(&o.Operation, &op)
		dto.Status.Unsuspension.Data = append(dto.Status.Unsuspension.Data, op)
	}
}
