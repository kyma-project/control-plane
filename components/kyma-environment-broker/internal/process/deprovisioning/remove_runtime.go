package deprovisioning

import (
	"fmt"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/hyperscaler"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
)

const (
	// the time after which the operation is marked as expired
	RemoveRuntimeTimeout = 1 * time.Hour
)

type RemoveRuntimeStep struct {
	operationManager  *process.DeprovisionOperationManager
	instanceStorage   storage.Instances
	provisionerClient provisioner.Client
	accountProvider   hyperscaler.AccountProvider
}

func NewRemoveRuntimeStep(os storage.Operations, is storage.Instances, cli provisioner.Client, accountProvider hyperscaler.AccountProvider) *RemoveRuntimeStep {
	return &RemoveRuntimeStep{
		operationManager:  process.NewDeprovisionOperationManager(os),
		instanceStorage:   is,
		provisionerClient: cli,
		accountProvider:   accountProvider,
	}
}

func (s *RemoveRuntimeStep) Name() string {
	return "Remove_Runtime"
}

func (s *RemoveRuntimeStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > RemoveRuntimeTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", RemoveRuntimeTimeout))
	}

	instance, err := s.instanceStorage.GetByID(operation.InstanceID)
	switch {
	case err == nil:
	case dberr.IsNotFound(err):
		return s.operationManager.OperationSucceeded(operation, "instance already deprovisioned")
	default:
		log.Errorf("unable to get instance from storage: %s", err)
		return operation, 1 * time.Second, nil
	}

	if instance.RuntimeID == "" {
		log.Warn("Runtime not exist")
		return operation, 0, nil
	}
	log = log.WithField("runtimeID", instance.RuntimeID)

	var provisionerResponse string
	if operation.ProvisionerOperationID == "" {

		//err := releaseSubscription (&operation)

		// mark subscription to be released with cleanup job if this is the
		// the only cluster for this GlobalAccountID (tenant)
		pp, err := operation.GetProvisioningParameters()
		if err != nil {
			// if the parameters are incorrect, there is no reason to retry the operation
			// a new request has to be issued by the user
			errorMessage := fmt.Sprintf("Aborting deprovisioning after failing to get valid operation provisioning parameters: %v", err)
			log.Errorf(errorMessage)
			return operation, 0, nil
		}

		if !broker.IsTrialPlan(pp.PlanID) {
			hypType, err := hyperscaler.HyperscalerTypeForPlanID(pp.PlanID)
			if err != nil {
				log.Errorf("Aborting deprovisioning after failing to determine the type of Hyperscaler to use for planID: %s", pp.PlanID)
				return operation, 0, nil
			}
			// combine both of them
			usedSubscriptions, err := s.accountProvider.GetNumberOfUsedSubscriptions(hypType, instance.GlobalAccountID, false)

			if err != nil {
				log.Errorf("Aborting deprovisioning after failing to determine number of used %s subscriptions by tenant: %s", hypType, instance.GlobalAccountID)
				return operation, 0, nil
			}

			if usedSubscriptions == 1 {
				s.accountProvider.ReleaseSubscription(hypType, instance.GlobalAccountID)
			}

		}


		provisionerResponse, err = s.provisionerClient.DeprovisionRuntime(instance.GlobalAccountID, instance.RuntimeID)
		if err != nil {
			log.Errorf("unable to deprovision runtime: %s", err)
			return operation, 10 * time.Second, nil
		}
		operation.ProvisionerOperationID = provisionerResponse
		log.Infof("fetched ProvisionerOperationID=%s", provisionerResponse)

		operation, repeat, err := s.operationManager.UpdateOperation(operation)
		if repeat != 0 {
			log.Errorf("cannot save operation ID from provisioner: %s", err)
			return operation, 5 * time.Second, nil
		}
	}

	log.Infof("runtime deletion process initiated successfully")
	// return repeat mode (1 sec) to start the initialization step which will now check the runtime status
	return operation, 1 * time.Second, nil
}

//func (s *RemoveRuntimeStep) releaseSubscription () {
//
//}
