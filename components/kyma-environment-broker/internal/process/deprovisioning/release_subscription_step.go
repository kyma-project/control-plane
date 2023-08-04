package deprovisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type ReleaseSubscriptionStep struct {
	operationManager *process.OperationManager
	instanceStorage  storage.Instances
	accountProvider  hyperscaler.AccountProvider
}

var _ process.Step = &ReleaseSubscriptionStep{}

func NewReleaseSubscriptionStep(os storage.Operations, instanceStorage storage.Instances, accountProvider hyperscaler.AccountProvider) ReleaseSubscriptionStep {
	return ReleaseSubscriptionStep{
		operationManager: process.NewOperationManager(os),
		instanceStorage:  instanceStorage,
		accountProvider:  accountProvider,
	}
}

func (s ReleaseSubscriptionStep) Name() string {
	return "Release_Subscription"
}

func (s ReleaseSubscriptionStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {

	planID := operation.ProvisioningParameters.PlanID
	if !broker.IsTrialPlan(planID) && !broker.IsOwnClusterPlan(planID) {
		instance, err := s.instanceStorage.GetByID(operation.InstanceID)
		if err != nil {
			msg := fmt.Sprintf("after successful deprovisioning failing to release hyperscaler subscription - get the instance data for instanceID [%s]: %s", operation.InstanceID, err.Error())
			operation, repeat, err := s.operationManager.MarkStepAsExcutedButNotCompleted(operation, s.Name(), msg, log)
			if repeat != 0 {
				return operation, repeat, err
			}
			return operation, 0, nil
		}

		if string(instance.Provider) == "" {
			log.Info("Instance does not contain cloud provider info due to failed provisioning, skipping")
			return operation, 0, nil
		}

		hypType, err := hyperscaler.FromCloudProvider(instance.Provider)
		if err != nil {
			msg := fmt.Sprintf("after successful deprovisioning failing to release hyperscaler subscription - determine the type of hyperscaler to use for planID [%s]: %s", planID, err.Error())
			operation, repeat, err := s.operationManager.MarkStepAsExcutedButNotCompleted(operation, s.Name(), msg, log)
			if repeat != 0 {
				return operation, repeat, err
			}
			return operation, 0, nil
		}

		euAccess := internal.IsEuAccess(operation.ProvisioningParameters.PlatformRegion)
		err = s.accountProvider.MarkUnusedGardenerSecretBindingAsDirty(hypType, instance.GetSubscriptionGlobalAccoundID(), euAccess)
		if err != nil {
			log.Errorf("after successful deprovisioning failed to release hyperscaler subscription: %s", err)
			return operation, 10 * time.Second, nil
		}
	}
	return operation, 0, nil
}
