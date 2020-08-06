package hyperscaler

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
)

//go:generate mockery -name=AccountProvider -output=automock -outpkg=automock -case=underscore
type AccountProvider interface {
	GardenerCredentials(hyperscalerType Type, tenantName string) (Credentials, error)
	GardenerSharedCredentials(hyperscalerType Type) (Credentials, error)
	GardenerSecretName(input *gqlschema.GardenerConfigInput, tenantName string) (string, error)
	ReleaseGardenerSecretForLastCluster(hyperscalerType Type, tenantName string) error
}

type accountProvider struct {
	compassPool        AccountPool
	gardenerPool       AccountPool
	sharedGardenerPool SharedPool
}

func NewAccountProvider(compassPool AccountPool, gardenerPool AccountPool, sharedGardenerPool SharedPool) AccountProvider {
	return &accountProvider{
		compassPool:        compassPool,
		gardenerPool:       gardenerPool,
		sharedGardenerPool: sharedGardenerPool,
	}
}

func HyperscalerTypeForPlanID(planID string) (Type, error) {

	switch planID {
	case broker.GCPPlanID, broker.GcpTrialPlanID:
		return GCP, nil
	case broker.AzurePlanID, broker.AzureLitePlanID, broker.AzureTrialPlanID:
		return Azure, nil
	default:
		return "", errors.Errorf("Cannot determine the type of Hyperscaler to use for planID: %s", planID)
	}

}

func HyperscalerTypeFromProvisionInput(input *gqlschema.ProvisionRuntimeInput) (Type, error) {

	if input == nil {
		return Type(""), errors.New("can't determine hyperscaler type because ProvisionRuntimeInput not specified (was nil)")
	}
	if input.ClusterConfig == nil {
		return Type(""), errors.New("can't determine hyperscaler type because ProvisionRuntimeInput.ClusterConfig not specified (was nil)")
	}

	if input.ClusterConfig.GardenerConfig == nil {
		return Type(""), errors.New("can't determine hyperscaler type because ProvisionRuntimeInput.ClusterConfig.GardenerConfig not specified (was nil)")
	}

	return HyperscalerTypeFromProviderString(input.ClusterConfig.GardenerConfig.Provider)
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

func (p *accountProvider) GardenerSecretName(input *gqlschema.GardenerConfigInput, tenantName string) (string, error) {
	if len(input.TargetSecret) > 0 {
		return input.TargetSecret, nil
	}

	hyperscalerType, err := HyperscalerTypeFromProviderString(input.Provider)
	if err != nil {
		return "", err
	}

	credential, err := p.GardenerCredentials(hyperscalerType, tenantName)
	if err != nil {
		return "", err
	}

	return credential.Name, nil
}

func (p *accountProvider) ReleaseGardenerSecretForLastCluster(hyperscalerType Type, tenantName string) error {
	if p.gardenerPool == nil {
		return errors.New("failed to release subscription for tenant. Gardener Account pool is not configured")
	}

	released, err := p.gardenerPool.IsSubscriptionAlreadyReleased(hyperscalerType, tenantName)

	if err != nil {
		return err
	}

	if released {
		return nil
	}

	usedSubscriptions, err := p.gardenerPool.CountSubscriptionUsages(hyperscalerType, tenantName)

	if err != nil {
		return errors.Wrapf(err, "Cannot determine number of used %s subscriptions by tenant: %s", hyperscalerType, tenantName)
	}

	if usedSubscriptions == 1 {
		p.gardenerPool.ReleaseSubscription(hyperscalerType, tenantName)
	}
	return nil
}
