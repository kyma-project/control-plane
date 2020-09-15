package runtime

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type converter struct {
	defaultSubaccountRegion string
}

func NewConverter(region string) *converter {
	return &converter{
		defaultSubaccountRegion: region,
	}
}

func (c *converter) getRegionOrDefault(instance internal.Instance) (string, error) {
	pp, err := instance.GetProvisioningParameters()
	if err != nil {
		return "", errors.Wrap(err, "while getting provisioning parameters")
	}

	if pp.PlatformRegion == "" {
		return c.defaultSubaccountRegion, nil
	}
	return pp.PlatformRegion, nil
}

func (c *converter) InstancesAndOperationsToDTO(instance internal.Instance, pOpr *internal.ProvisioningOperation,
	dOpr *internal.DeprovisioningOperation, ukOpr *internal.UpgradeKymaOperation) (runtimeDTO, error) {
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
			CreatedAt:    instance.CreatedAt,
			ModifiedAt:   instance.UpdatedAt,
			Provisioning: &operation{},
		},
	}

	region, err := c.getRegionOrDefault(instance)
	if err != nil {
		return runtimeDTO{}, errors.Wrap(err, "while getting region")
	}
	toReturn.SubAccountRegion = region

	urlSplitted := strings.Split(instance.DashboardURL, ".")
	if len(urlSplitted) > 1 {
		toReturn.ShootName = urlSplitted[1]
	}

	if pOpr != nil {
		toReturn.Status.Provisioning.State = string(pOpr.State)
		toReturn.Status.Provisioning.Description = pOpr.Description
	}
	if dOpr != nil {

	}
	if ukOpr != nil {
		toReturn.Status.UpgradingKyma = &operation{
			State:       string(ukOpr.State),
			Description: ukOpr.Description,
		}
	}

	return toReturn, nil
}
