package broker

import (
	"encoding/json"
	"strings"

	"github.com/kyma-incubator/compass/components/director/pkg/jsonschema"
	"github.com/pkg/errors"

	"github.com/pivotal-cf/brokerapi/v8/domain"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

const (
	AllPlansSelector = "all_plans"

	GCPPlanID         = "ca6e5357-707f-4565-bbbd-b3ab732597c6"
	GCPPlanName       = "gcp"
	AWSPlanID         = "361c511f-f939-4621-b228-d0fb79a1fe15"
	AWSPlanName       = "aws"
	AWSHAPlanID       = "aecef2e6-49f1-4094-8433-eba0e135eb6a"
	AWSHAPlanName     = "aws_ha"
	AzurePlanID       = "4deee563-e5ec-4731-b9b1-53b42d855f0c"
	AzurePlanName     = "azure"
	AzureLitePlanID   = "8cb22518-aa26-44c5-91a0-e669ec9bf443"
	AzureLitePlanName = "azure_lite"
	AzureHAPlanID     = "f2951649-02ca-43a5-9188-9c07fb612491"
	AzureHAPlanName   = "azure_ha"
	TrialPlanID       = "7d55d31d-35ae-4438-bf13-6ffdfa107d9f"
	TrialPlanName     = "trial"
	OpenStackPlanID   = "03b812ac-c991-4528-b5bd-08b303523a63"
	OpenStackPlanName = "openstack"
	FreemiumPlanID    = "b1a5764e-2ea1-4f95-94c0-2b4538b37b55"
	FreemiumPlanName  = "free"
	PreviewPlanID     = "5cb3d976-b85c-42ea-a636-79cadda109a9"
	PreviewPlanName   = "preview"
)

var PlanNamesMapping = map[string]string{
	GCPPlanID:       GCPPlanName,
	AWSPlanID:       AWSPlanName,
	AWSHAPlanID:     AWSHAPlanName,
	AzurePlanID:     AzurePlanName,
	AzureLitePlanID: AzureLitePlanName,
	AzureHAPlanID:   AzureHAPlanName,
	TrialPlanID:     TrialPlanName,
	OpenStackPlanID: OpenStackPlanName,
	FreemiumPlanID:  FreemiumPlanName,
	PreviewPlanID:   PreviewPlanName,
}

var PlanIDsMapping = map[string]string{
	AzurePlanName:     AzurePlanID,
	AWSPlanName:       AWSPlanID,
	AWSHAPlanName:     AWSHAPlanID,
	AzureLitePlanName: AzureLitePlanID,
	AzureHAPlanName:   AzureHAPlanID,
	GCPPlanName:       GCPPlanID,
	TrialPlanName:     TrialPlanID,
	OpenStackPlanName: OpenStackPlanID,
	FreemiumPlanName:  FreemiumPlanID,
	PreviewPlanName:   PreviewPlanID,
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

func AWSHARegions() []string {
	// be aware of zones defined in internal/provider/aws_provider.go
	return []string{"eu-central-1", "eu-west-2", "ca-central-1", "sa-east-1", "us-east-1",
		"ap-northeast-1", "ap-northeast-2", "ap-south-1", "ap-southeast-1", "ap-southeast-2"}
}

func OpenStackRegions() []string {
	return []string{"eu-de-1", "ap-sa-1"}
}

func OpenStackSchema(machineTypes []string) RootSchema {
	properties := NewProvisioningProperties(machineTypes, OpenStackRegions())
	return NewSchema(properties, DefaultControlsOrder())
}

func GCPSchema(machineTypes []string) RootSchema {
	properties := NewProvisioningProperties(machineTypes, GCPRegions())
	return NewSchema(properties, DefaultControlsOrder())
}

func AWSSchema(machineTypes []string) RootSchema {
	properties := NewProvisioningProperties(machineTypes, AWSRegions())
	return NewSchema(properties, DefaultControlsOrder())
}

func AWSHASchema(machineTypes []string) RootSchema {
	properties := NewProvisioningProperties(machineTypes, AWSHARegions())
	properties.ZonesCount = &Type{
		Type:        "integer",
		Minimum:     3,
		Maximum:     3,
		Default:     3,
		Description: "Specifies the number of availability zones for HA cluster",
	}
	awsHaControlsOrder := DefaultControlsOrder()
	awsHaControlsOrder = append(awsHaControlsOrder, "zonesCount")
	schema := NewSchema(properties, awsHaControlsOrder)

	properties.AutoScalerMin.Default = 1
	properties.AutoScalerMin.Minimum = 1
	properties.AutoScalerMin.Description = "Specifies the minimum number of virtual machines to create per zone"

	properties.AutoScalerMax.Minimum = 1
	properties.AutoScalerMax.Description = "Specifies the maximum number of virtual machines to create per zone"

	return schema
}

func AzureSchema(machineTypes []string) RootSchema {
	properties := NewProvisioningProperties(machineTypes, AzureRegions())
	return NewSchema(properties, DefaultControlsOrder())
}

func AzureLiteSchema(machineTypes []string) RootSchema {
	properties := NewProvisioningProperties(machineTypes, AzureRegions())
	properties.AutoScalerMax.Maximum = 40
	properties.AutoScalerMax.Default = 10

	return NewSchema(properties, DefaultControlsOrder())
}

func FreemiumSchema(provider internal.CloudProvider) RootSchema {
	var regions []string
	switch provider {
	case internal.AWS:
		regions = AWSRegions()
	case internal.Azure:
		regions = AzureRegions()
	default:
		regions = AWSRegions()
	}
	schema := NewSchema(
		ProvisioningProperties{
			Name: NameProperty(),
			Region: &Type{
				Type: "string",
				Enum: ToInterfaceSlice(regions),
			},
		}, []string{"name", "region"})

	return schema
}

func AzureHASchema(machineTypes []string) RootSchema {
	properties := NewProvisioningProperties(machineTypes, AzureRegions())
	properties.ZonesCount = &Type{
		Type:        "integer",
		Minimum:     3,
		Maximum:     3,
		Default:     3,
		Description: "Specifies the number of availability zones for HA cluster",
	}
	azureHaControlsOrder := DefaultControlsOrder()
	azureHaControlsOrder = append(azureHaControlsOrder, "zonesCount")
	schema := NewSchema(properties, azureHaControlsOrder)

	properties.AutoScalerMin.Default = 1
	properties.AutoScalerMin.Minimum = 1
	properties.AutoScalerMin.Description = "Specifies the minimum number of virtual machines to create per zone"

	properties.AutoScalerMax.Minimum = 1
	properties.AutoScalerMax.Description = "Specifies the maximum number of virtual machines to create per zone"

	return schema
}

func TrialSchema() RootSchema {
	return NewSchema(
		ProvisioningProperties{
			Name: NameProperty(),
		}, DefaultControlsOrder())
}

func marshalSchema(schema RootSchema) []byte {
	bytes, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}
	return bytes
}

func schemaForUpdate(provisioningRoot RootSchema) []byte {
	pp := provisioningRoot.Properties.(ProvisioningProperties)
	if pp.AutoScalerMax == nil && pp.AutoScalerMin == nil && pp.OIDC == nil && pp.Administrators == nil {
		return []byte{}
	}
	up := UpdateProperties{}
	if pp.AutoScalerMax != nil {
		up.AutoScalerMax = &Type{
			Minimum:     pp.AutoScalerMax.Minimum,
			Maximum:     pp.AutoScalerMax.Maximum,
			Description: pp.AutoScalerMax.Description,
			Type:        pp.AutoScalerMax.Type,
		}
	}
	if pp.AutoScalerMin != nil {
		up.AutoScalerMin = &Type{
			Minimum:     pp.AutoScalerMin.Minimum,
			Maximum:     pp.AutoScalerMin.Maximum,
			Description: pp.AutoScalerMin.Description,
			Type:        pp.AutoScalerMin.Type,
		}
	}
	if pp.OIDC != nil {
		up.OIDC = pp.OIDC
	}
	if pp.Administrators != nil {
		up.Administrators = pp.Administrators
	}

	return marshalSchema(NewUpdateSchema(up))
}

// Plan is a wrapper for OSB API ServicePlan
type Plan struct {
	PlanDefinition domain.ServicePlan
	// catalogRawSchema is JSONSchema which is exposed on /v2/catalog endpoint - if empty, provisioningRawSchema is used
	catalogRawSchema []byte
	// provisioningRawSchema is a JSONSchema which serves as validation source for provisioning input
	provisioningRawSchema []byte
	updateRawSchema       []byte
}

// Plans is designed to hold plan defaulting logic
// keep internal/hyperscaler/azure/config.go in sync with any changes to available zones
func Plans(plans PlansConfig, provider internal.CloudProvider, includeAdditionalParamsInSchema bool) map[string]Plan {
	awsSchema := AWSSchema([]string{"m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge", "m6i.2xlarge", "m6i.4xlarge", "m6i.8xlarge", "m6i.12xlarge"})
	awsHASchema := AWSHASchema([]string{"m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge", "m6i.2xlarge", "m6i.4xlarge", "m6i.8xlarge", "m6i.12xlarge"})
	gcpSchema := GCPSchema([]string{"n2-standard-8", "n2-standard-16", "n2-standard-32", "n2-standard-48"})
	openstackSchema := OpenStackSchema([]string{"m2.xlarge", "m1.2xlarge"})
	azureSchema := AzureSchema([]string{"Standard_D8_v3"})
	azureLiteSchema := AzureLiteSchema([]string{"Standard_D4_v3"})
	azureHASchema := AzureHASchema([]string{"Standard_D8_v3"})
	freemiumSchema := FreemiumSchema(provider)
	trialSchema := TrialSchema()

	// Schemas exposed on v2/catalog endpoint - different than provisioningRawSchema to allow backwards compatibility
	// when a machine type switch is introduced
	// switch to m6 if m6 is available in all regions
	awsCatalogSchema := AWSSchema([]string{"m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge"})
	awsHACatalogSchema := AWSHASchema([]string{"m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge"})

	if includeAdditionalParamsInSchema {
		schemas := []*RootSchema{
			&awsSchema,
			&awsCatalogSchema,
			&awsHASchema,
			&awsHACatalogSchema,
			&gcpSchema,
			&openstackSchema,
			&azureSchema,
			&azureLiteSchema,
			&azureHASchema,
			&freemiumSchema,
			&trialSchema,
		}
		includeOIDCSchema(schemas...)
		includeAdminsSchema(schemas...)
	}

	outputPlans := map[string]Plan{
		AWSPlanID: {
			PlanDefinition:        defaultServicePlan(AWSPlanID, AWSHAPlanName, plans),
			catalogRawSchema:      marshalSchema(awsCatalogSchema),
			provisioningRawSchema: marshalSchema(awsSchema),
			updateRawSchema:       schemaForUpdate(awsSchema),
		},
		PreviewPlanID: {
			PlanDefinition:        defaultServicePlan(PreviewPlanID, PreviewPlanName, plans),
			provisioningRawSchema: marshalSchema(awsSchema),
			updateRawSchema:       schemaForUpdate(awsSchema),
		},
		AWSHAPlanID: {
			PlanDefinition:        defaultServicePlan(AWSHAPlanID, AWSHAPlanName, plans),
			catalogRawSchema:      marshalSchema(awsHACatalogSchema),
			provisioningRawSchema: marshalSchema(awsHASchema),
			updateRawSchema:       schemaForUpdate(awsHASchema),
		},
		GCPPlanID: {
			PlanDefinition:        defaultServicePlan(GCPPlanID, GCPPlanName, plans),
			provisioningRawSchema: marshalSchema(gcpSchema),
			updateRawSchema:       schemaForUpdate(gcpSchema),
		},
		OpenStackPlanID: {
			PlanDefinition:        defaultServicePlan(OpenStackPlanID, OpenStackPlanName, plans),
			provisioningRawSchema: marshalSchema(openstackSchema),
			updateRawSchema:       schemaForUpdate(openstackSchema),
		},
		AzurePlanID: {
			PlanDefinition:        defaultServicePlan(AzurePlanID, AzurePlanName, plans),
			provisioningRawSchema: marshalSchema(azureSchema),
			updateRawSchema:       schemaForUpdate(azureSchema),
		},
		AzureLitePlanID: {
			PlanDefinition:        defaultServicePlan(AzureLitePlanID, AzureHAPlanID, plans),
			provisioningRawSchema: marshalSchema(azureLiteSchema),
			updateRawSchema:       schemaForUpdate(azureLiteSchema),
		},
		FreemiumPlanID: {
			PlanDefinition:        defaultServicePlan(FreemiumPlanID, FreemiumPlanName, plans),
			provisioningRawSchema: marshalSchema(freemiumSchema),
			updateRawSchema:       schemaForUpdate(freemiumSchema),
		},
		AzureHAPlanID: {
			PlanDefinition:        defaultServicePlan(AzureHAPlanID, AzureHAPlanName, plans),
			provisioningRawSchema: marshalSchema(azureHASchema),
			updateRawSchema:       schemaForUpdate(azureHASchema),
		},
		TrialPlanID: {
			PlanDefinition:        defaultServicePlan(TrialPlanID, TrialPlanName, plans),
			provisioningRawSchema: marshalSchema(trialSchema),
			updateRawSchema:       schemaForUpdate(trialSchema),
		},
	}

	// for _, plan := range outputPlans {
	// 	plan.
	// }
	// b.updateControlsOrder(&p.Schemas.Instance)

	return outputPlans
}

func includeOIDCSchema(schemas ...*RootSchema) {
	oidcSchema := NewOIDCSchema()

	for _, schema := range schemas {
		pp := schema.Properties.(ProvisioningProperties)
		pp.OIDC = &oidcSchema
		schema.Properties = pp
		schema.ControlsOrder = append(schema.ControlsOrder, "oidc")
	}
}

func includeAdminsSchema(schemas ...*RootSchema) {
	adminsProperty := AdministratorsProperty()

	for _, schema := range schemas {
		pp := schema.Properties.(ProvisioningProperties)
		pp.Administrators = adminsProperty
		schema.Properties = pp
		schema.ControlsOrder = append(schema.ControlsOrder, "administrators")
	}
}

func defaultServicePlan(id, name string, plans PlansConfig) domain.ServicePlan {
	return domain.ServicePlan{
		ID:          id,
		Name:        name,
		Description: defaultDescription(name, plans),
		Metadata:    defaultMetadata(name, plans), Schemas: &domain.ServiceSchemas{
			Instance: domain.ServiceInstanceSchema{
				Create: domain.Schema{
					Parameters: make(map[string]interface{}),
				},
				Update: domain.Schema{
					Parameters: make(map[string]interface{}),
				},
			},
		},
	}
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

func IsPreviewPlan(planID string) bool {
	switch planID {
	case PreviewPlanID:
		return true
	default:
		return false
	}
}

func updateControlsOrder(schema *domain.ServiceInstanceSchema) error {
	casted, ok := schema.Create.Parameters[ControlsOrderKey].([]interface{})
	if !ok {
		return errors.New("Invalid type of Create _controlsOrder param")
	}

	targetControls, err := appendNonExisting(make(map[string]int), casted)
	if err != nil {
		return errors.Wrap(err, "Error while creating _controlsOrder")
	}

	casted, ok = schema.Update.Parameters[ControlsOrderKey].([]interface{})
	if !ok {
		return errors.New("Invalid type of Update _controlsOrder param")
	}

	targetControls, err = appendNonExisting(targetControls, casted)
	if err != nil {
		return errors.Wrap(err, "Error while creating _controlsOrder")
	}

	inverted := invert(targetControls)

	createProps := schema.Create.Parameters[PropertiesKey].(map[string]interface{})
	schema.Create.Parameters[ControlsOrderKey], err =
		filterAndOrder(inverted, createProps)
	if err != nil {
		return errors.New("Error while updating Create controlOrder")
	}

	updateProps := schema.Update.Parameters[PropertiesKey].(map[string]interface{})
	schema.Update.Parameters[ControlsOrderKey], err =
		filterAndOrder(inverted, updateProps)
	if err != nil {
		return errors.New("Error while updating Update controlOrder")
	}

	return nil
}

func appendNonExisting(to map[string]int, from []interface{}) (map[string]int, error) {
	size := len(to)
	for i, v := range from {
		key, ok := v.(string)

		if !ok {
			return nil, errors.Errorf("Invalid value type")
		}

		if _, ok = to[key]; !ok {
			to[key] = size + i
		}
	}

	return to, nil
}

func invert(targetControls map[string]int) []string {
	inverted := make([]string, len(targetControls))

	for key, value := range targetControls {
		inverted[value] = key
	}

	return inverted
}

func filterAndOrder(items []string, included map[string]interface{}) ([]interface{}, error) {
	output := make([]interface{}, 0)
	for i := 0; i < len(items); i++ {
		value := items[i]

		if _, ok := included[value]; ok {
			output = append(output, value)
		}
	}

	return output, nil
}
