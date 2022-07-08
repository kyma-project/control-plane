package broker

import (
	"strings"

	"github.com/kyma-incubator/compass/components/director/pkg/jsonschema"

	"github.com/pivotal-cf/brokerapi/v8/domain"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

const (
	AllPlansSelector = "all_plans"

	GCPPlanID         = "ca6e5357-707f-4565-bbbd-b3ab732597c6"
	GCPPlanName       = "gcp"
	AWSPlanID         = "361c511f-f939-4621-b228-d0fb79a1fe15"
	AWSPlanName       = "aws"
	AzurePlanID       = "4deee563-e5ec-4731-b9b1-53b42d855f0c"
	AzurePlanName     = "azure"
	AzureLitePlanID   = "8cb22518-aa26-44c5-91a0-e669ec9bf443"
	AzureLitePlanName = "azure_lite"
	TrialPlanID       = "7d55d31d-35ae-4438-bf13-6ffdfa107d9f"
	TrialPlanName     = "trial"
	OpenStackPlanID   = "03b812ac-c991-4528-b5bd-08b303523a63"
	OpenStackPlanName = "openstack"
	FreemiumPlanID    = "b1a5764e-2ea1-4f95-94c0-2b4538b37b55"
	FreemiumPlanName  = "free"
)

var PlanNamesMapping = map[string]string{
	GCPPlanID:       GCPPlanName,
	AWSPlanID:       AWSPlanName,
	AzurePlanID:     AzurePlanName,
	AzureLitePlanID: AzureLitePlanName,
	TrialPlanID:     TrialPlanName,
	OpenStackPlanID: OpenStackPlanName,
	FreemiumPlanID:  FreemiumPlanName,
}

var PlanIDsMapping = map[string]string{
	AzurePlanName:     AzurePlanID,
	AWSPlanName:       AWSPlanID,
	AzureLitePlanName: AzureLitePlanID,
	GCPPlanName:       GCPPlanID,
	TrialPlanName:     TrialPlanID,
	OpenStackPlanName: OpenStackPlanID,
	FreemiumPlanName:  FreemiumPlanID,
}

type TrialCloudRegion string

const (
	Europe TrialCloudRegion = "europe"
	Us     TrialCloudRegion = "us"
	Asia   TrialCloudRegion = "asia"
)

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

func OpenStackSchema(machineTypes []string, additionalParams, update bool) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypes, OpenStackRegions(), update)
	// Specifying multiple zones for openstack runtimes are not supported.
	properties.ZonesCount = nil
	properties.AutoScalerMax.Maximum = 40
	if !update {
		properties.AutoScalerMax.Default = 8
	}
	return createSchemaWithProperties(properties, additionalParams, update)
}

func GCPSchema(machineTypes []string, additionalParams, update bool) *map[string]interface{} {
	return createSchema(machineTypes, GCPRegions(), additionalParams, update)
}

func AWSSchema(machineTypes []string, additionalParams, update bool) *map[string]interface{} {
	return createSchema(machineTypes, AWSRegions(), additionalParams, update)
}

func AzureSchema(machineTypes []string, additionalParams, update bool) *map[string]interface{} {
	return createSchema(machineTypes, AzureRegions(), additionalParams, update)
}

func AzureLiteSchema(machineTypes []string, additionalParams, update bool) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypes, AzureRegions(), update)
	properties.ZonesCount = nil
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

func createSchema(machineTypes, regions []string, additionalParams, update bool) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypes, regions, update)
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

	// awsHASchema := AWSHASchema(awsMachines, includeAdditionalParamsInSchema, false)

	gcpMachines := []string{"n2-standard-4", "n2-standard-8", "n2-standard-16", "n2-standard-32", "n2-standard-48"}
	gcpSchema := GCPSchema(gcpMachines, includeAdditionalParamsInSchema, false)

	openStackMachines := []string{"g_c4_m16", "g_c8_m32"}
	openstackSchema := OpenStackSchema(openStackMachines, includeAdditionalParamsInSchema, false)

	azureMachines := []string{"Standard_D4_v3", "Standard_D8_v3"}
	azureSchema := AzureSchema(azureMachines, includeAdditionalParamsInSchema, false)

	azureLiteSchema := AzureLiteSchema([]string{"Standard_D4_v3"}, includeAdditionalParamsInSchema, false)
	freemiumSchema := FreemiumSchema(provider, includeAdditionalParamsInSchema, false)
	trialSchema := TrialSchema(includeAdditionalParamsInSchema, false)

	// Schemas exposed on v2/catalog endpoint - different than provisioningRawSchema to allow backwards compatibility
	// when a machine type switch is introduced
	// switch to m6 if m6 is available in all regions
	awsCatalogMachines := []string{"m5.xlarge", "m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge"}
	awsCatalogSchema := AWSSchema(awsCatalogMachines, includeAdditionalParamsInSchema, false)

	outputPlans := map[string]domain.ServicePlan{
		AWSPlanID:       defaultServicePlan(AWSPlanID, AWSPlanName, plans, awsCatalogSchema, AWSSchema(awsMachines, includeAdditionalParamsInSchema, true)),
		GCPPlanID:       defaultServicePlan(GCPPlanID, GCPPlanName, plans, gcpSchema, GCPSchema(gcpMachines, includeAdditionalParamsInSchema, true)),
		OpenStackPlanID: defaultServicePlan(OpenStackPlanID, OpenStackPlanName, plans, openstackSchema, OpenStackSchema(openStackMachines, includeAdditionalParamsInSchema, true)),
		AzurePlanID:     defaultServicePlan(AzurePlanID, AzurePlanName, plans, azureSchema, AzureSchema(azureMachines, includeAdditionalParamsInSchema, true)),
		AzureLitePlanID: defaultServicePlan(AzureLitePlanID, AzureLitePlanName, plans, azureLiteSchema, AzureLiteSchema([]string{"Standard_D4_v3"}, includeAdditionalParamsInSchema, true)),
		FreemiumPlanID:  defaultServicePlan(FreemiumPlanID, FreemiumPlanName, plans, freemiumSchema, FreemiumSchema(provider, includeAdditionalParamsInSchema, true)),
		TrialPlanID:     defaultServicePlan(TrialPlanID, TrialPlanName, plans, trialSchema, TrialSchema(includeAdditionalParamsInSchema, true)),
	}

	return outputPlans
}

func defaultServicePlan(id, name string, plans PlansConfig, createParams, updateParams *map[string]interface{}) domain.ServicePlan {
	servicePlan := domain.ServicePlan{
		ID:          id,
		Name:        name,
		Description: defaultDescription(name, plans),
		Metadata:    defaultMetadata(name, plans), Schemas: &domain.ServiceSchemas{
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
