package deprovisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type ReleaseSubscriptionStep struct {
	instanceStorage storage.Instances
	accountProvider hyperscaler.AccountProvider
}

var _ process.Step = &ReleaseSubscriptionStep{}

func NewReleaseSubscriptionStep(instanceStorage storage.Instances, accountProvider hyperscaler.AccountProvider) ReleaseSubscriptionStep {
	return ReleaseSubscriptionStep{
		instanceStorage: instanceStorage,
		accountProvider: accountProvider,
	}
}

func (s ReleaseSubscriptionStep) Name() string {
	return "Release_credentials_secret_binding"
}

func (s ReleaseSubscriptionStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {

	planID := operation.ProvisioningParameters.PlanID
	if !broker.IsTrialPlan(planID) {
		instance, err := s.instanceStorage.GetByID(operation.InstanceID)
		if err != nil {
			log.Errorf("after successful deprovisioning failing to release hyperscaler subscription - get the instance data for instanceID: %s", operation.InstanceID, err.Error())
			return operation, 0, nil
		}

		hypType, err := hyperscaler.FromCloudProvider(instance.Provider)
		if err != nil {
			log.Errorf("after successful deprovisioning failing to release hyperscaler subscription - determine the type of hyperscaler to use for planID [%s]: %s", planID, err.Error())
			return operation, 0, nil
		}

		err = s.accountProvider.MarkUnusedGardenerSecretBindingAsDirty(hypType, instance.GetSubscriptionGlobalAccoundID())
		if err != nil {
			log.Errorf("after successful deprovisioning failed to release hyperscaler subscription: %s", err)
			return operation, 10 * time.Second, nil
		}
	}
	return operation, 0, nil
}
