package provisioning

import (
	"context"
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/azure"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	processazure "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/azure"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"
)

// ensure the interface is implemented
var _ Step = (*ProvisionAzureResourceGroupStep)(nil)

type ProvisionAzureResourceGroupStep struct {
	operationManager *process.ProvisionOperationManager
	azure            processazure.ProviderContext
}

func NewProvisionAzureResourceGroupStep(os storage.Operations, hyperscalerProvider azure.HyperscalerProvider, accountProvider hyperscaler.AccountProvider, ctx context.Context) *ProvisionAzureResourceGroupStep {
	return &ProvisionAzureResourceGroupStep{
		operationManager: process.NewProvisionOperationManager(os),
		azure: processazure.ProviderContext{
			HyperscalerProvider: hyperscalerProvider,
			AccountProvider:     accountProvider,
			Context:             ctx,
		},
	}
}

func (s *ProvisionAzureResourceGroupStep) Name() string {
	return "Provision Azure Resource Group"
}

func (s *ProvisionAzureResourceGroupStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	// check if step was finished successfully before, and the resource group name was persisted
	if operation.Azure.ResourceGroupName != "" {
		log.Info("Resource Group is already provisioned")
		return operation, 0, nil
	}

	hypType := hyperscaler.Azure

	// parse provisioning parameters
	pp, err := operation.GetProvisioningParameters()
	if err != nil {
		// if the parameters are incorrect, there is no reason to retry the operation
		// a new request has to be issued by the user
		log.Errorf("Aborting after failing to get valid operation provisioning parameters: %v", err)
		return s.operationManager.OperationFailed(operation, "invalid operation provisioning parameters")
	}

	// get hyperscaler credentials from HAP
	var credentials hyperscaler.Credentials
	if !broker.IsTrialPlan(pp.PlanID) {
		log.Infof("HAP lookup for credentials to provision Resource Group for global account ID %s on Hyperscaler %s", pp.ErsContext.GlobalAccountID, hypType)
		credentials, err = s.azure.AccountProvider.GardenerCredentials(hypType, pp.ErsContext.GlobalAccountID)
	} else {
		log.Infof("HAP lookup for shared credentials to provision Resource Group on Hyperscaler %s", hypType)
		credentials, err = s.azure.AccountProvider.GardenerSharedCredentials(hypType)
	}
	if err != nil {
		// retrying might solve the issue, the HAP could be temporarily unavailable
		errorMessage := fmt.Sprintf("Unable to retrieve Gardener Credentials from HAP lookup: %v", err)
		return s.operationManager.RetryOperation(operation, errorMessage, time.Minute, time.Minute*30, log)
	}
	azureCfg, err := azure.GetConfigFromHAPCredentialsAndProvisioningParams(credentials, pp)
	if err != nil {
		// internal error, repeating doesn't solve the problem
		errorMessage := fmt.Sprintf("Failed to create Azure config: %v", err)
		return s.operationManager.OperationFailed(operation, errorMessage)
	}

	// create hyperscaler client
	azureClient, err := s.azure.HyperscalerProvider.GetClient(azureCfg, log)
	if err != nil {
		// internal error, repeating doesn't solve the problem
		errorMessage := fmt.Sprintf("Failed to create Azure client: %v", err)
		return s.operationManager.OperationFailed(operation, errorMessage)
	}

	// prepare azure tags
	tags := azure.Tags{
		azure.TagSubAccountID: &pp.ErsContext.SubAccountID,
		azure.TagInstanceID:   &operation.InstanceID,
		azure.TagOperationID:  &operation.ID,
	}

	// prepare a valid unique name for Azure resources
	uniqueName := processazure.GetAzureResourceName(operation.InstanceID)

	// create Resource Group
	groupName := uniqueName
	_, err = azureClient.CreateResourceGroup(s.azure.Context, azureCfg, groupName, tags)
	if err != nil {
		// retrying might solve the issue while communicating with azure, e.g. network problems etc
		errorMessage := fmt.Sprintf("Failed to persist Azure Resource Group [%s] with error: %v", groupName, err)
		return s.operationManager.RetryOperation(operation, errorMessage, time.Minute, time.Minute*30, log)
	}
	log.Printf("Persisted Azure Resource Group [%s]", groupName)

	// persist the created resource group name
	operation.Azure.ResourceGroupName = groupName
	op, repeat := s.operationManager.UpdateOperation(operation)
	if repeat != 0 {
		log.Errorf("cannot save Azure Resource Group name")
		return operation, time.Second, nil
	}

	return op, 0, nil
}
