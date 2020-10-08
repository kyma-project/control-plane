package runtime

import (
	"strings"

	pkg "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/pkg/errors"
)

type converter struct {
	defaultSubaccountRegion string
}

func NewConverter(platformRegion string) *converter {
	return &converter{
		defaultSubaccountRegion: platformRegion,
	}
}

func (c *converter) setRegionOrDefault(instance internal.Instance, runtime *pkg.RuntimeDTO) error {
	pp, err := instance.GetProvisioningParameters()
	if err != nil {
		return errors.Wrap(err, "while getting provisioning parameters")
	}

	if pp.PlatformRegion == "" {
		runtime.SubAccountRegion = c.defaultSubaccountRegion
	} else {
		runtime.SubAccountRegion = pp.PlatformRegion
	}
	if pp.Parameters.Region != nil {
		runtime.ProviderRegion = *pp.Parameters.Region
	} else {
		runtime.ProviderRegion = ""
	}
	return nil
}

func (c *converter) InstancesAndOperationsToDTO(instance internal.Instance, pOpr *internal.ProvisioningOperation,
	dOpr *internal.DeprovisioningOperation, ukOpr *internal.UpgradeKymaOperation) (pkg.RuntimeDTO, error) {
	toReturn := pkg.RuntimeDTO{
		InstanceID:       instance.InstanceID,
		RuntimeID:        instance.RuntimeID,
		GlobalAccountID:  instance.GlobalAccountID,
		SubAccountID:     instance.SubAccountID,
		ServiceClassID:   instance.ServiceID,
		ServiceClassName: instance.ServiceName,
		ServicePlanID:    instance.ServicePlanID,
		ServicePlanName:  instance.ServicePlanName,
		Status: pkg.RuntimeStatus{
			CreatedAt:    instance.CreatedAt,
			ModifiedAt:   instance.UpdatedAt,
			Provisioning: &pkg.Operation{},
		},
	}

	err := c.setRegionOrDefault(instance, &toReturn)
	if err != nil {
		return pkg.RuntimeDTO{}, errors.Wrap(err, "while setting region")
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
		toReturn.Status.Deprovisioning = &pkg.Operation{
			State:       string(dOpr.State),
			Description: dOpr.Description,
		}
	}
	if ukOpr != nil {
		toReturn.Status.UpgradingKyma = &pkg.Operation{
			State:       string(ukOpr.State),
			Description: ukOpr.Description,
		}
	}

	return toReturn, nil
}
