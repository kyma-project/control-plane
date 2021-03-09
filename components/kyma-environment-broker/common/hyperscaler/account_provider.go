package hyperscaler

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/pkg/errors"
)

//go:generate mockery -name=AccountProvider -output=automock -outpkg=automock -case=underscore
type AccountProvider interface {
	GardenerCredentials(hyperscalerType Type, tenantName string) (Credentials, error)
	GardenerSharedCredentials(hyperscalerType Type) (Credentials, error)
	MarkUnusedGardenerSecretBindingAsDirty(hyperscalerType Type, tenantName string) error
}

type accountProvider struct {
	gardenerPool       AccountPool
	sharedGardenerPool SharedPool
}

func NewAccountProvider(gardenerPool AccountPool, sharedGardenerPool SharedPool) AccountProvider {
	return &accountProvider{
		gardenerPool:       gardenerPool,
		sharedGardenerPool: sharedGardenerPool,
	}
}

func HyperscalerTypeForPlanID(planID string) (Type, error) {

	switch planID {
	case broker.GCPPlanID:
		return GCP, nil
	case broker.AzurePlanID, broker.AzureLitePlanID:
		return Azure, nil
	default:
		return "", errors.Errorf("cannot determine the type of Hyperscaler to use for planID: %s", planID)
	}
}

func (p *accountProvider) GardenerCredentials(hyperscalerType Type, tenantName string) (Credentials, error) {

	if p.gardenerPool == nil {
		return Credentials{},
			errors.New("failed to get Gardener Credentials. Gardener Account pool is not configured")
	}

	return p.gardenerPool.Credentials(hyperscalerType, tenantName)
}

func (p *accountProvider) GardenerSharedCredentials(hyperscalerType Type) (Credentials, error) {
	if p.sharedGardenerPool == nil {
		return Credentials{},
			errors.New("failed to get shared Gardener Credentials. Gardener Shared Account pool is not configured")
	}

	return p.sharedGardenerPool.SharedCredentials(hyperscalerType)
}

func (p *accountProvider) MarkUnusedGardenerSecretBindingAsDirty(hyperscalerType Type, tenantName string) error {
	if p.gardenerPool == nil {
		return errors.New("failed to release subscription for tenant. Gardener Account pool is not configured")
	}

	internal, err := p.gardenerPool.IsSecretBindingInternal(hyperscalerType, tenantName)
	if err != nil {
		return err
	}

	if internal {
		return nil
	}

	dirty, err := p.gardenerPool.IsSecretBindingDirty(hyperscalerType, tenantName)
	if err != nil {
		return err
	}

	if dirty {
		return nil
	}

	secretBindingUsed, err := p.gardenerPool.IsSecretBindingUsed(hyperscalerType, tenantName)

	if err != nil {
		return errors.Wrapf(err, "cannot determine whether %s secret binding is used for tenant: %s", hyperscalerType, tenantName)
	}

	if !secretBindingUsed {
		return p.gardenerPool.MarkSecretBindingAsDirty(hyperscalerType, tenantName)
	}

	return nil
}
