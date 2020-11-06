package provisioning

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/azure"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	processazure "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/azure"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/uid"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
)

const (
	registryNamePrefix = "kyma"
	registryAddress    = "azurecr.io"
)

var (
	registryNameRegexp = regexp.MustCompile(`[^a-zA-Z0-9]`)
)

//go:generate mockery -name=UIDGenerator -output=automock -outpkg=automock -case=underscore
type UIDGenerator interface {
	Generate() string
}

// ensure the interface is implemented
var _ Step = (*ProvisionAzureContainerRegistryStep)(nil)

type ProvisionAzureContainerRegistryStep struct {
	operationManager *process.ProvisionOperationManager
	azure            processazure.ProviderContext
	azureStepConfig  azure.StepConfig
	uidSvc           UIDGenerator
}

func NewProvisionAzureContainerRegistryStep(os storage.Operations, hyperscalerProvider azure.HyperscalerProvider, accountProvider hyperscaler.AccountProvider, stepCfg azure.StepConfig, ctx context.Context) *ProvisionAzureContainerRegistryStep {
	return &ProvisionAzureContainerRegistryStep{
		operationManager: process.NewProvisionOperationManager(os),
		azure: processazure.ProviderContext{
			HyperscalerProvider: hyperscalerProvider,
			AccountProvider:     accountProvider,
			Context:             ctx,
		},
		azureStepConfig: stepCfg,
		uidSvc:          uid.NewUIDService(),
	}
}

func (s *ProvisionAzureContainerRegistryStep) Name() string {
	return "Provision Azure Container Registry"
}

func (s *ProvisionAzureContainerRegistryStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	// check if step was finished successfully before, and the resource group name was persisted
	if operation.Azure.ContainerRegistryCreated {
		log.Info("Container Registry is already provisioned")
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
		log.Infof("HAP lookup for credentials to provision Container Registry for global account ID %s on Hyperscaler %s", pp.ErsContext.GlobalAccountID, hypType)
		credentials, err = s.azure.AccountProvider.GardenerCredentials(hypType, pp.ErsContext.GlobalAccountID)
	} else {
		log.Infof("HAP lookup for shared credentials to provision Container Registry on Hyperscaler %s", hypType)
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
	uniqueRegistryName := s.getValidRegistryName()

	// retrieve azure resource group name from operation
	groupName := operation.Azure.ResourceGroupName
	if groupName == "" {
		// resource group wasn't correctly created in previous steps
		errorMessage := fmt.Sprintf("Failed to retrieve name of Azure Resource Group")
		return s.operationManager.RetryOperation(operation, errorMessage, time.Minute, time.Minute*30, log)
	}

	// create Container Registry
	_, err = azureClient.CreateContainerRegistry(s.azure.Context, azureCfg, uniqueRegistryName, groupName, tags, s.azureStepConfig.ContainerRegistrySKU)
	if err != nil {
		// retrying might solve the issue while communicating with azure, e.g. network problems etc
		errorMessage := fmt.Sprintf("Failed to persist Azure Container Registry [%s] with error: %v", uniqueRegistryName, err)
		return s.operationManager.RetryOperation(operation, errorMessage, time.Minute, time.Minute*30, log)
	}

	// retrieve password to newly created registry
	registryCredentials, err := azureClient.ListContainerRegistryCredentials(s.azure.Context, uniqueRegistryName, groupName)
	if err != nil {
		// retrying might solve the issue while communicating with azure, e.g. network problems etc
		errorMessage := fmt.Sprintf("Failed to retrieve Azure Container Registry credentials for registry [%s] with error: %v", uniqueRegistryName, err)
		return s.operationManager.RetryOperation(operation, errorMessage, time.Minute, time.Minute*30, log)
	}
	if registryCredentials.Passwords == nil || (*registryCredentials.Passwords)[0].Value == nil {
		// if ListContainerRegistryCredentials() does not fail then a non-nil accessKey is returned
		// then retry the operation once
		errorMessage := "Passwords is nil"
		return s.operationManager.RetryOperationOnce(operation, errorMessage, time.Second*15, log)
	}
	registryPassword := (*registryCredentials.Passwords)[0].Value

	// apply overrides
	operation.InputCreator.AppendOverrides(components.Serverless, s.getServerlessOverrides(uniqueRegistryName, *registryPassword))

	// persist the state of provisioning
	operation.Azure.ContainerRegistryCreated = true
	op, repeat := s.operationManager.UpdateOperation(operation)
	if repeat != 0 {
		log.Errorf("cannot save Azure Container Registry provisioning state")
		return operation, time.Second, nil
	}

	return op, 0, nil
}

func (s *ProvisionAzureContainerRegistryStep) getServerlessOverrides(username, password string) []*gqlschema.ConfigEntryInput {
	return []*gqlschema.ConfigEntryInput{
		{
			Key:    "dockerRegistry.enableInternal",
			Value:  "false",
			Secret: to.BoolPtr(true),
		},
		{
			Key:    "dockerRegistry.username",
			Value:  username,
			Secret: to.BoolPtr(true),
		},
		{
			Key:    "dockerRegistry.password",
			Value:  password,
			Secret: to.BoolPtr(true),
		},
		{
			Key:    "dockerRegistry.serverAddress",
			Value:  fmt.Sprintf("%s.%s", strings.ToLower(username), registryAddress),
			Secret: to.BoolPtr(true),
		},
		{
			Key:    "dockerRegistry.registryAddress",
			Value:  fmt.Sprintf("%s.%s", strings.ToLower(username), registryAddress),
			Secret: to.BoolPtr(true),
		},
	}
}

// getValidRegistryName returns a valid unique Azure Container Registry name.
// The name must be a 5-50 characters long alphanumeric string.
// https://docs.microsoft.com/en-us/azure/container-registry/container-registry-get-started-azure-cli#create-a-container-registry
func (s *ProvisionAzureContainerRegistryStep) getValidRegistryName() string {
	name := fmt.Sprintf("%s%s", registryNamePrefix, s.uidSvc.Generate())
	name = registryNameRegexp.ReplaceAllString(name, "")
	return name
}
