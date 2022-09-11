package broker

import (
	"strings"

	"github.com/kyma-incubator/compass/components/director/pkg/jsonschema"

	"github.com/pivotal-cf/brokerapi/v8/domain"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

const (
	AllPlansSelector = "all_plans"

	GCPPlanID          = "ca6e5357-707f-4565-bbbd-b3ab732597c6"
	GCPPlanName        = "gcp"
	AWSPlanID          = "361c511f-f939-4621-b228-d0fb79a1fe15"
	AWSPlanName        = "aws"
	AzurePlanID        = "4deee563-e5ec-4731-b9b1-53b42d855f0c"
	AzurePlanName      = "azure"
	AzureLitePlanID    = "8cb22518-aa26-44c5-91a0-e669ec9bf443"
	AzureLitePlanName  = "azure_lite"
	TrialPlanID        = "7d55d31d-35ae-4438-bf13-6ffdfa107d9f"
	TrialPlanName      = "trial"
	OpenStackPlanID    = "03b812ac-c991-4528-b5bd-08b303523a63"
	OpenStackPlanName  = "openstack"
	FreemiumPlanID     = "b1a5764e-2ea1-4f95-94c0-2b4538b37b55"
	FreemiumPlanName   = "free"
	OwnClusterPlanID   = "03e3cb66-a4c6-4c6a-b4b0-5d42224debea"
	OwnClusterPlanName = "owncluster"
)

var PlanNamesMapping = map[string]string{
	GCPPlanID:        GCPPlanName,
	AWSPlanID:        AWSPlanName,
	AzurePlanID:      AzurePlanName,
	AzureLitePlanID:  AzureLitePlanName,
	TrialPlanID:      TrialPlanName,
	OpenStackPlanID:  OpenStackPlanName,
	FreemiumPlanID:   FreemiumPlanName,
	OwnClusterPlanID: OwnClusterPlanName,
}

var PlanIDsMapping = map[string]string{
	AzurePlanName:      AzurePlanID,
	AWSPlanName:        AWSPlanID,
	AzureLitePlanName:  AzureLitePlanID,
	GCPPlanName:        GCPPlanID,
	TrialPlanName:      TrialPlanID,
	OpenStackPlanName:  OpenStackPlanID,
	FreemiumPlanName:   FreemiumPlanID,
	OwnClusterPlanName: OwnClusterPlanID,
}

type TrialCloudRegion string

const (
	Europe TrialCloudRegion = "europe"
	Us     TrialCloudRegion = "us"
	Asia   TrialCloudRegion = "asia"
)

var validRegionsForTrial = map[TrialCloudRegion]struct{}{
	Europe: {},
	Us:     {},
	Asia:   {},
}

type JSONSchemaValidator interface {
	ValidateString(json string) (jsonschema.ValidationResult, error)
}

func AzureRegions() []string {
	return []string{
		"eastus",
		"centralus",
		"westus2",
		"uksouth",
		"northeurope",
		"westeurope",
		"japaneast",
		"southeastasia",
	}
}

func GCPRegions() []string {
	return []string{
		"europe-west3",
		"asia-south1",
		"us-central1"}
}

func AWSRegions() []string {
	// be aware of zones defined in internal/provider/aws_provider.go
	return []string{"eu-central-1", "eu-west-2", "ca-central-1", "sa-east-1", "us-east-1", "us-west-1",
		"ap-northeast-1", "ap-northeast-2", "ap-south-1", "ap-southeast-1", "ap-southeast-2"}
}

func OpenStackRegions() []string {
	return []string{"eu-de-1", "ap-sa-1"}
}

func OpenStackSchema(machineTypesDisplay map[string]string, machineTypes []string, additionalParams, update bool) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypesDisplay, machineTypes, OpenStackRegions(), update)
	properties.AutoScalerMax.Maximum = 40
	if !update {
		properties.AutoScalerMax.Default = 8
	}

	return createSchemaWithProperties(properties, additionalParams, update)
}

func GCPSchema(machineTypesDisplay map[string]string, machineTypes []string, additionalParams, update bool) *map[string]interface{} {
	return createSchema(machineTypesDisplay, machineTypes, GCPRegions(), additionalParams, update)
}

func AWSSchema(machineTypesDisplay map[string]string, machineTypes []string, additionalParams, update bool) *map[string]interface{} {
	return createSchema(machineTypesDisplay, machineTypes, AWSRegions(), additionalParams, update)
}

func AzureSchema(machineTypesDisplay map[string]string, machineTypes []string, additionalParams, update bool) *map[string]interface{} {
	return createSchema(machineTypesDisplay, machineTypes, AzureRegions(), additionalParams, update)
}

func AzureLiteSchema(machineTypesDisplay map[string]string, machineTypes []string, additionalParams, update bool) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypesDisplay, machineTypes, AzureRegions(), update)
	properties.AutoScalerMax.Maximum = 40

	if !update {
		properties.AutoScalerMax.Default = 10
		properties.AutoScalerMin.Default = 2
	}

	return createSchemaWithProperties(properties, additionalParams, update)
}

func FreemiumSchema(provider internal.CloudProvider, additionalParams, update bool) *map[string]interface{} {
	if update && !additionalParams {
		return empty()
	}

	var regions []string
	switch provider {
	case internal.AWS:
		regions = AWSRegions()
	case internal.Azure:
		regions = AzureRegions()
	default:
		regions = AWSRegions()
	}
	properties := ProvisioningProperties{
		Name: NameProperty(),
		Region: &Type{
			Type: "string",
			Enum: ToInterfaceSlice(regions),
		},
	}

	return createSchemaWithProperties(properties, additionalParams, update)
}

func TrialSchema(additionalParams, update bool) *map[string]interface{} {
	properties := ProvisioningProperties{
		Name: NameProperty(),
	}

	if update && !additionalParams {
		return empty()
	}

	return createSchemaWithProperties(properties, additionalParams, update)
}

func empty() *map[string]interface{} {
	empty := make(map[string]interface{}, 0)
	return &empty
}

func createSchema(machineTypesDisplay map[string]string, machineTypes, regions []string, additionalParams, update bool) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypesDisplay, machineTypes, regions, update)
	return createSchemaWithProperties(properties, additionalParams, update)
}

func createSchemaWithProperties(properties ProvisioningProperties, additionalParams, update bool) *map[string]interface{} {
	if additionalParams {
		properties.IncludeAdditional()
	}

	if update {
		return createSchemaWith(properties.UpdateProperties, update)
	} else {
		return createSchemaWith(properties, update)
	}
}

func createSchemaWith(properties interface{}, update bool) *map[string]interface{} {
	schema := NewSchema(properties, update)

	target := make(map[string]interface{})
	schema.ControlsOrder = DefaultControlsOrder()

	unmarshaled := unmarshalOrPanic(schema, &target).(*map[string]interface{})

	// update controls order
	props := (*unmarshaled)[PropertiesKey].(map[string]interface{})
	controlsOrder := (*unmarshaled)[ControlsOrderKey].([]interface{})
	(*unmarshaled)[ControlsOrderKey] = filter(&controlsOrder, props)

	return unmarshaled
}

// Plans is designed to hold plan defaulting logic
// keep internal/hyperscaler/azure/config.go in sync with any changes to available zones
func Plans(plans PlansConfig, provider internal.CloudProvider, includeAdditionalParamsInSchema bool) map[string]domain.ServicePlan {
	awsMachines := []string{"m5.xlarge", "m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge", "m6i.xlarge", "m6i.2xlarge", "m6i.4xlarge", "m6i.8xlarge", "m6i.12xlarge"}
	awsMachinesDisplay := map[string]string{
		// source: https://aws.amazon.com/ec2/instance-types/m5/
		"m5.xlarge":   "m5.xlarge (4vCPU, 16GB RAM)",
		"m5.2xlarge":  "m5.2xlarge (8vCPU, 32GB RAM)",
		"m5.4xlarge":  "m5.4xlarge (16vCPU, 64GB RAM)",
		"m5.8xlarge":  "m5.8xlarge (32vCPU, 128GB RAM)",
		"m5.12xlarge": "m5.12xlarge (48vCPU, 192GB RAM)",
		// source: https://aws.amazon.com/ec2/instance-types/m6i/
		"m6i.xlarge":   "m6i.xlarge (4vCPU, 16GB RAM)",
		"m6i.2xlarge":  "m6i.2xlarge (8vCPU, 32GB RAM)",
		"m6i.4xlarge":  "m6i.4xlarge (16vCPU, 64GB RAM)",
		"m6i.8xlarge":  "m6i.8xlarge (32vCPU, 128GB RAM)",
		"m6i.12xlarge": "m6i.12xlarge (48vCPU, 192GB RAM)",
	}

	// awsHASchema := AWSHASchema(awsMachinesDisplay, awsMachines, includeAdditionalParamsInSchema, false)

	// source: https://cloud.google.com/compute/docs/general-purpose-machines#e2_limitations
	gcpMachines := []string{"n2-standard-4", "n2-standard-8", "n2-standard-16", "n2-standard-32", "n2-standard-48"}
	gcpMachinesDisplay := map[string]string{
		"n2-standard-4":  "n2-standard-4 (4vCPU, 16GB RAM)",
		"n2-standard-8":  "n2-standard-8 (8vCPU, 32GB RAM)",
		"n2-standard-16": "n2-standard-16 (16vCPU, 64GB RAM)",
		"n2-standard-32": "n2-standard-32 (32vCPU, 128GB RAM)",
		"n2-standard-48": "n2-standard-48 (48vCPU, 192B RAM)",
	}
	gcpSchema := GCPSchema(gcpMachinesDisplay, gcpMachines, includeAdditionalParamsInSchema, false)

	openStackMachines := []string{"g_c4_m16", "g_c8_m32"}
	openStackMachinesDisplay := map[string]string{
		"g_c4_m16": "g_c4_m16 (4vCPU, 16GB RAM)",
		"g_c8_m32": "g_c8_m32 (8vCPU, 32GB RAM)",
	}
	openstackSchema := OpenStackSchema(openStackMachinesDisplay, openStackMachines, includeAdditionalParamsInSchema, false)

	// source: https://docs.microsoft.com/en-us/azure/cloud-services/cloud-services-sizes-specs#dv3-series
	azureMachines := []string{"Standard_D4_v3", "Standard_D8_v3", "Standard_D16_v3", "Standard_D32_v3", "Standard_D48_v3", "Standard_D64_v3"}
	azureMachinesDisplay := map[string]string{
		"Standard_D4_v3":  "Standard_D4_v3 (4vCPU, 16GB RAM)",
		"Standard_D8_v3":  "Standard_D8_v3 (8vCPU, 32GB RAM)",
		"Standard_D16_v3": "Standard_D16_v3 (16vCPU, 64GB RAM)",
		"Standard_D32_v3": "Standard_D32_v3 (32vCPU, 128GB RAM)",
		"Standard_D48_v3": "Standard_D48_v3 (48vCPU, 192GB RAM)",
		"Standard_D64_v3": "Standard_D64_v3 (64vCPU, 256GB RAM)",
	}
	azureSchema := AzureSchema(azureMachinesDisplay, azureMachines, includeAdditionalParamsInSchema, false)

	azureLiteMachines := []string{"Standard_D4_v3"}
	azureLiteMachinesDisplay := map[string]string{
		"Standard_D4_v3": azureMachinesDisplay["Standard_D4_v3"],
	}
	azureLiteSchema := AzureLiteSchema(azureLiteMachinesDisplay, azureLiteMachines, includeAdditionalParamsInSchema, false)
	freemiumSchema := FreemiumSchema(provider, includeAdditionalParamsInSchema, false)
	trialSchema := TrialSchema(includeAdditionalParamsInSchema, false)

	// Schemas exposed on v2/catalog endpoint - different than provisioningRawSchema to allow backwards compatibility
	// when a machine type switch is introduced
	// switch to m6 if m6 is available in all regions
	awsCatalogMachines := []string{"m5.xlarge", "m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge"}
	awsCatalogMachinesDisplay := map[string]string{
		"m5.xlarge":   awsMachinesDisplay["m5.xlarge"],
		"m5.2xlarge":  awsMachinesDisplay["m5.2xlarge"],
		"m5.4xlarge":  awsMachinesDisplay["m5.4xlarge"],
		"m5.8xlarge":  awsMachinesDisplay["m5.8xlarge"],
		"m5.12xlarge": awsMachinesDisplay["m5.12xlarge"],
	}
	awsCatalogSchema := AWSSchema(awsCatalogMachinesDisplay, awsCatalogMachines, includeAdditionalParamsInSchema, false)

	outputPlans := map[string]domain.ServicePlan{
		AWSPlanID:       defaultServicePlan(AWSPlanID, AWSPlanName, plans, awsCatalogSchema, AWSSchema(awsMachinesDisplay, awsMachines, includeAdditionalParamsInSchema, true)),
		GCPPlanID:       defaultServicePlan(GCPPlanID, GCPPlanName, plans, gcpSchema, GCPSchema(gcpMachinesDisplay, gcpMachines, includeAdditionalParamsInSchema, true)),
		OpenStackPlanID: defaultServicePlan(OpenStackPlanID, OpenStackPlanName, plans, openstackSchema, OpenStackSchema(openStackMachinesDisplay, openStackMachines, includeAdditionalParamsInSchema, true)),
		AzurePlanID:     defaultServicePlan(AzurePlanID, AzurePlanName, plans, azureSchema, AzureSchema(azureMachinesDisplay, azureMachines, includeAdditionalParamsInSchema, true)),
		AzureLitePlanID: defaultServicePlan(AzureLitePlanID, AzureLitePlanName, plans, azureLiteSchema, AzureLiteSchema(azureLiteMachinesDisplay, azureLiteMachines, includeAdditionalParamsInSchema, true)),
		FreemiumPlanID:  defaultServicePlan(FreemiumPlanID, FreemiumPlanName, plans, freemiumSchema, FreemiumSchema(provider, includeAdditionalParamsInSchema, true)),
		TrialPlanID:     defaultServicePlan(TrialPlanID, TrialPlanName, plans, trialSchema, TrialSchema(includeAdditionalParamsInSchema, true)),
    OwnClusterPlanID: defaultServicePlan(OwnClusterPlanID, OwnClusterPlanName, plans, trialSchema, TrialSchema(includeAdditionalParamsInSchema, true)),
	}

	return outputPlans
}

func defaultServicePlan(id, name string, plans PlansConfig, createParams, updateParams *map[string]interface{}) domain.ServicePlan {
	servicePlan := domain.ServicePlan{
		ID:          id,
		Name:        name,
		Description: defaultDescription(name, plans),
		Metadata:    defaultMetadata(name, plans),
		Schemas: &domain.ServiceSchemas{
			Instance: domain.ServiceInstanceSchema{
				Create: domain.Schema{
					Parameters: *createParams,
				},
				Update: domain.Schema{
					Parameters: *updateParams,
				},
			},
		},
	}

	return servicePlan
}

func defaultDescription(planName string, plans PlansConfig) string {
	plan, ok := plans[planName]
	if !ok || len(plan.Description) == 0 {
		return strings.ToTitle(planName)
	}

	return plan.Description
}
func defaultMetadata(planName string, plans PlansConfig) *domain.ServicePlanMetadata {
	plan, ok := plans[planName]
	if !ok || len(plan.Metadata.DisplayName) == 0 {
		return &domain.ServicePlanMetadata{
			DisplayName: strings.ToTitle(planName),
		}
	}
	return &domain.ServicePlanMetadata{
		DisplayName: plan.Metadata.DisplayName,
	}
}

func IsTrialPlan(planID string) bool {
	switch planID {
	case TrialPlanID:
		return true
	default:
		return false
	}
}

func IsAzurePlan(planID string) bool {
	switch planID {
	case AzurePlanID, AzureLitePlanID:
		return true
	default:
		return false
	}
}

func IsFreemiumPlan(planID string) bool {
	switch planID {
	case FreemiumPlanID:
		return true
	default:
		return false
	}
}

func filter(items *[]interface{}, included map[string]interface{}) interface{} {
	output := make([]interface{}, 0)
	for i := 0; i < len(*items); i++ {
		value := (*items)[i]

		if _, ok := included[value.(string)]; ok {
			output = append(output, value)
		}
	}

	return output
}
