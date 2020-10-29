package deprovisioning

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/azure"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	processazure "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/azure"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

type DeprovisionAzureResourceGroupStep struct {
	OperationManager *process.DeprovisionOperationManager
	processazure.ProviderContext
}

func NewDeprovisionAzureResourceGroupStep(os storage.Operations, hyperscalerProvider azure.HyperscalerProvider,
	accountProvider hyperscaler.AccountProvider,
	ctx context.Context) DeprovisionAzureResourceGroupStep {
	return DeprovisionAzureResourceGroupStep{
		OperationManager: process.NewDeprovisionOperationManager(os),
		ProviderContext: processazure.ProviderContext{
			HyperscalerProvider: hyperscalerProvider,
			AccountProvider:     accountProvider,
			Context:             ctx,
		},
	}
}

var _ Step = (*DeprovisionAzureResourceGroupStep)(nil)

func (s DeprovisionAzureResourceGroupStep) Name() string {
	return "Deprovision Azure Resource Group"
}

func (s DeprovisionAzureResourceGroupStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (
	internal.DeprovisioningOperation, time.Duration, error) {
	if operation.Azure.ResourceGroupDeleted {
		log.Info("Resource Group is already deprovisioned")
		return operation, 0, nil
	}

	hypType := hyperscaler.Azure

	pp, err := operation.GetProvisioningParameters()
	if err != nil {
		// if the parameters are incorrect, there is no reason to retry the operation
		// a new request has to be issued by the user
		errorMessage := fmt.Sprintf("aborting deprovisioning after failing to get valid operation provisioning"+
			" parameters: %v", err)
		log.Errorf(errorMessage)
		return operation, 0, nil
	}
	log.Infof("HAP lookup for credentials to deprovision cluster for global account ID %s on Hyperscaler %s",
		pp.ErsContext.GlobalAccountID, hypType)

	//get hyperscaler credentials from HAP
	credentials, err := s.ProviderContext.AccountProvider.GardenerCredentials(hypType, pp.ErsContext.GlobalAccountID)
	if err != nil {
		// retrying might solve the issue, the HAP could be temporarily unavailable
		errorMessage := fmt.Sprintf("unable to retrieve Gardener Credentials from HAP lookup: %v", err)
		return s.OperationManager.RetryOperationWithoutFail(operation, errorMessage, time.Minute, 30*time.Minute, log)
	}
	azureCfg, err := azure.GetConfigFromHAPCredentialsAndProvisioningParams(credentials, pp)
	if err != nil {
		// internal error, repeating doesn't solve the problem
		log.Errorf("failed to create Azure config: %v", err)
		return operation, 0, nil
	}

	// create hyperscaler client
	namespaceClient, err := s.HyperscalerProvider.GetClient(azureCfg, log)
	if err != nil {
		// internal error, repeating doesn't solve the problem
		log.Errorf("failed to create Azure client: %v", err)
		return operation, 0, nil
	}
	// prepare azure tags
	tags := azure.Tags{azure.TagInstanceID: &operation.InstanceID}

	// check if resource group exists
	resourceGroup, err := namespaceClient.GetResourceGroup(s.ProviderContext.Context, tags)
	if err != nil {
		// if it doesn't exist anymore, there is nothing to delete - we are done
		if _, ok := err.(azure.ResourceGroupDoesNotExistError); ok {
			if &resourceGroup != nil && resourceGroup.Name != nil {
				log.Infof("deprovisioning of resource group step succeeded, resource group: %v", resourceGroup.Name)
			} else {
				log.Info("deprovisioning of resource group step succeeded")
			}
			operation.Azure.ResourceGroupDeleted = true
			// save new operation state to database
			op, _, err := s.OperationManager.UpdateOperation(operation)
			if err != nil {
				log.Error("cannot save Azure Resource Group deprovisioning state")
				return operation, time.Second, nil
			}
			return op, 0, nil
		}
		// custom error occurred while getting resource group - try again
		errorMessage := fmt.Sprintf("error while getting resource group, error: %v", err)
		return s.OperationManager.RetryOperationWithoutFail(operation, errorMessage, time.Minute, time.Hour, log)
	}
	// delete the resource group if it still exists and deletion has not been triggered yet
	if resourceGroup.Properties == nil || resourceGroup.Properties.ProvisioningState == nil {
		log.Errorf("nil pointer in the resource group")
		return operation, 0, nil
	}
	deprovisioningState := *resourceGroup.Properties.ProvisioningState
	if deprovisioningState != azure.FutureOperationDeleting {
		future, err := namespaceClient.DeleteResourceGroup(s.ProviderContext.Context, tags)
		if err != nil {
			errorMessage := fmt.Sprintf("unable to delete Azure resource group: %v", err)
			return s.OperationManager.RetryOperationWithoutFail(operation, errorMessage, time.Minute, time.Hour,
				log)
		}
		if future.Status() != azure.FutureOperationSucceeded {
			var retryAfterDuration time.Duration
			if retryAfter, retryAfterHeaderExist := future.GetPollingDelay(); retryAfterHeaderExist {
				retryAfterDuration = retryAfter
			} else {
				retryAfterDuration = time.Minute
			}
			log.Infof("rescheduling step to check deletion of resource group completed after %v",
				retryAfterDuration)
			return s.OperationManager.RetryOperationWithoutFail(operation,
				"waiting for deprovisioning of azure resource group", retryAfterDuration, time.Hour, log)
		}
	}
	errorMessage := "waiting for deprovisioning of azure resource group"
	return s.OperationManager.RetryOperationWithoutFail(operation, errorMessage, time.Minute, time.Hour, log)
}
