package provisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ResolveCredentialsStep struct {
	operationManager *process.ProvisionOperationManager
	accountProvider  hyperscaler.AccountProvider
	opStorage        storage.Operations
	tenant           string
}

func getHyperscalerType(pp internal.ProvisioningParameters) (hyperscaler.Type, error) {
	switch pp.PlanID {
	case broker.GCPPlanID:
		return hyperscaler.GCP, nil
	case broker.AWSPlanID:
		return hyperscaler.AWS, nil
	case broker.AzurePlanID, broker.AzureLitePlanID, broker.AzureHAPlanID:
		return hyperscaler.Azure, nil
	case broker.OpenStackPlanID:
		return hyperscaler.Openstack, nil
	case broker.TrialPlanID:
		return forTrialProvider(pp.Parameters.Provider)
	default:
		return "", errors.Errorf("Cannot determine the type of Hyperscaler to use for planID: %s", pp.PlanID)
	}
}

func forTrialProvider(provider *internal.TrialCloudProvider) (hyperscaler.Type, error) {
	if provider == nil {
		return hyperscaler.Azure, nil
	}

	switch *provider {
	case internal.Azure:
		return hyperscaler.Azure, nil
	case internal.AWS:
		return hyperscaler.AWS, nil
	case internal.Gcp:
		return hyperscaler.GCP, nil
	default:
		return "", errors.Errorf("Cannot determine the type of Hyperscaler to use for provider: %s", string(*provider))
	}

}

func NewResolveCredentialsStep(os storage.Operations, accountProvider hyperscaler.AccountProvider) *ResolveCredentialsStep {
	return &ResolveCredentialsStep{
		operationManager: process.NewProvisionOperationManager(os),
		opStorage:        os,
		accountProvider:  accountProvider,
	}
}

func (s *ResolveCredentialsStep) Name() string {
	return "Resolve_Target_Secret"
}

func (s *ResolveCredentialsStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if operation.ProvisioningParameters.Parameters.TargetSecret != nil {
		return operation, 0, nil
	}

	hypType, err := getHyperscalerType(operation.ProvisioningParameters)
	if err != nil {
		log.Error("Aborting after failing to determine the type of Hyperscaler to use for planID: %s", operation.ProvisioningParameters.PlanID)
		return s.operationManager.OperationFailed(operation, err.Error(), log)
	}

	log.Infof("HAP lookup for credentials to provision cluster for global account ID %s on Hyperscaler %s", operation.ProvisioningParameters.ErsContext.GlobalAccountID, hypType)

	var credentials hyperscaler.Credentials
	if !broker.IsTrialPlan(operation.ProvisioningParameters.PlanID) {
		credentials, err = s.accountProvider.GardenerCredentials(hypType, operation.ProvisioningParameters.ErsContext.GlobalAccountID)
	} else {
		log.Infof("HAP lookup for shared credentials")
		credentials, err = s.accountProvider.GardenerSharedCredentials(hypType)
	}
	if err != nil {
		errMsg := fmt.Sprintf("HAP lookup for credentials to provision cluster for global account ID %s on Hyperscaler %s has failed: %s", operation.ProvisioningParameters.ErsContext.GlobalAccountID, hypType, err)
		log.Info(errMsg)

		// if failed retry step every 10s by next 10min
		dur := time.Since(operation.UpdatedAt).Round(time.Minute)

		if dur < 10*time.Minute {
			return operation, 10 * time.Second, nil
		}

		log.Errorf("Aborting after 10 minutes of failing to resolve provisioning credentials for global account ID %s on Hyperscaler %s", operation.ProvisioningParameters.ErsContext.GlobalAccountID, hypType)
		return s.operationManager.OperationFailed(operation, errMsg, log)
	}
	operation.ProvisioningParameters.Parameters.TargetSecret = &credentials.Name

	updatedOperation, err := s.opStorage.UpdateProvisioningOperation(operation)
	if err != nil {
		return operation, 1 * time.Minute, nil
	}

	log.Infof("Resolved %s as target secret name to use for cluster provisioning for global account ID %s on Hyperscaler %s", *operation.ProvisioningParameters.Parameters.TargetSecret, operation.ProvisioningParameters.ErsContext.GlobalAccountID, hypType)

	return *updatedOperation, 0, nil
}
