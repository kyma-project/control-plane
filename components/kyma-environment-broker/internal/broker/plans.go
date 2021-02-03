package broker

import (
	"encoding/json"

	"github.com/pivotal-cf/brokerapi/v7/domain"
)

const (
	AllPlansSelector = "all_plans"

	GCPPlanID         = "ca6e5357-707f-4565-bbbd-b3ab732597c6"
	GCPPlanName       = "gcp"
	AzurePlanID       = "4deee563-e5ec-4731-b9b1-53b42d855f0c"
	AzurePlanName     = "azure"
	AzureLitePlanID   = "8cb22518-aa26-44c5-91a0-e669ec9bf443"
	AzureLitePlanName = "azure_lite"
	TrialPlanID       = "7d55d31d-35ae-4438-bf13-6ffdfa107d9f"
	TrialPlanName     = "trial"
)

var PlanNamesMapping = map[string]string{
	GCPPlanID:       GCPPlanName,
	AzurePlanID:     AzurePlanName,
	AzureLitePlanID: AzureLitePlanName,
	TrialPlanID:     TrialPlanName,
}

var PlanIDsMapping = map[string]string{
	AzurePlanName:     AzurePlanID,
	AzureLitePlanName: AzureLitePlanID,
	GCPPlanName:       GCPPlanID,
	TrialPlanName:     TrialPlanID,
}

type TrialCloudRegion string

const (
	Europe TrialCloudRegion = "europe"
	Us     TrialCloudRegion = "us"
	Asia   TrialCloudRegion = "asia"
)

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
		"asia-south1", "asia-southeast1",
		"asia-east2", "asia-east1",
		"asia-northeast1", "asia-northeast2", "asia-northeast-3",
		"australia-southeast1",
		"europe-west2", "europe-west4", "europe-west5", "europe-west6", "europe-west3",
		"europe-north1",
		"us-west1", "us-west2", "us-west3",
		"us-central1",
		"us-east4",
		"northamerica-northeast1", "southamerica-east1"}
}

type Type struct {
	Type            string        `json:"type"`
	Title           string        `json:"title,omitempty"`
	Description     string        `json:"description,omitempty"`
	Minimum         int           `json:"minimum,omitempty"`
	Maximum         int           `json:"maximum,omitempty"`
	MinLength       int           `json:"minLength,omitempty"`
	MaxLength       int           `json:"maxLength,omitempty"`
	Default         interface{}   `json:"default,omitempty"`
	Example         interface{}   `json:"example,omitempty"`
	Enum            []interface{} `json:"enum,omitempty"`
	Items           []Type        `json:"items,omitempty"`
	AdditionalItems *bool         `json:"additionalItems,omitempty"`
	UniqueItems     *bool         `json:"uniqueItems,omitempty"`
}

type RootSchema struct {
	Schema string `json:"$schema"`
	Type
	Properties interface{} `json:"properties"`
	Required   []string    `json:"required"`

	// Specified to true enables form view on website
	ShowFormView bool `json:"_show_form_view"`
}

type ProvisioningProperties struct {
	Name          Type  `json:"name"`
	Region        *Type `json:"region,omitempty"`
	MachineType   *Type `json:"machineType,omitempty"`
	AutoScalerMin *Type `json:"autoScalerMin,omitempty"`
	AutoScalerMax *Type `json:"autoScalerMax,omitempty"`
}

func NameProperty() Type {
	return Type{
		Type:        "string",
		MinLength:   1,
		Title:       "Cluster Name",
		Description: "Specifies the name of the cluster",
	}
}

// NewProvisioningProperties creates a new properties for different plans
// Note that the order of properties will be the same in the form on the website
func NewProvisioningProperties(machineTypes []string, regions []string) ProvisioningProperties {
	return ProvisioningProperties{
		Name: NameProperty(),
		Region: &Type{
			Type:        "string",
			Enum:        ToInterfaceSlice(regions),
			Description: "Defines the cluster region",
		},
		MachineType: &Type{
			Type:        "string",
			Enum:        ToInterfaceSlice(machineTypes),
			Description: "Specifies the provider-specific virtual machine type",
		},
		AutoScalerMin: &Type{
			Type:        "integer",
			Minimum:     2,
			Default:     2,
			Description: "Specifies the minimum number of virtual machines to create",
		},
		AutoScalerMax: &Type{
			Type:        "integer",
			Minimum:     2,
			Maximum:     40,
			Default:     10,
			Description: "Specifies the maximum number of virtual machines to create",
		},
	}
}

func NewSchema(properties ProvisioningProperties) RootSchema {
	return RootSchema{
		Schema: "http://json-schema.org/draft-04/schema#",
		Type: Type{
			Type: "object",
		},
		Properties:   properties,
		Required:     []string{"name"},
		ShowFormView: true,
	}
}

func GCPSchema(machineTypes []string) []byte {
	schema := NewSchema(NewProvisioningProperties(machineTypes, GCPRegions()))

	bytes, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}
	return bytes
}

func AzureSchema(machineTypes []string) []byte {
	schema := NewSchema(NewProvisioningProperties(machineTypes, AzureRegions()))

	bytes, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}
	return bytes
}

func TrialSchema() []byte {
	schema := NewSchema(
		ProvisioningProperties{
			Name: NameProperty(),
		})

	bytes, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}
	return bytes
}

func ToInterfaceSlice(input []string) []interface{} {
	interfaces := make([]interface{}, len(input))
	for i, item := range input {
		interfaces[i] = item
	}
	return interfaces
}

// plans is designed to hold plan defaulting logic
// keep internal/hyperscaler/azure/config.go in sync with any changes to available zones
var Plans = map[string]struct {
	PlanDefinition        domain.ServicePlan
	provisioningRawSchema []byte
}{
	GCPPlanID: {
		PlanDefinition: domain.ServicePlan{
			ID:          GCPPlanID,
			Name:        GCPPlanName,
			Description: "GCP",
			Metadata: &domain.ServicePlanMetadata{
				DisplayName: "GCP",
			},
			Schemas: &domain.ServiceSchemas{
				Instance: domain.ServiceInstanceSchema{
					Create: domain.Schema{
						Parameters: make(map[string]interface{}),
					},
				},
			},
		},
		provisioningRawSchema: GCPSchema([]string{"n1-standard-2", "n1-standard-4", "n1-standard-8", "n1-standard-16", "n1-standard-32", "n1-standard-64"}),
	},
	AzurePlanID: {
		PlanDefinition: domain.ServicePlan{
			ID:          AzurePlanID,
			Name:        AzurePlanName,
			Description: "Azure",
			Metadata: &domain.ServicePlanMetadata{
				DisplayName: "Azure",
			},
			Schemas: &domain.ServiceSchemas{
				Instance: domain.ServiceInstanceSchema{
					Create: domain.Schema{
						Parameters: make(map[string]interface{}),
					},
				},
			},
		},
		provisioningRawSchema: AzureSchema([]string{"Standard_D8_v3"}),
	},
	AzureLitePlanID: {
		PlanDefinition: domain.ServicePlan{
			ID:          AzureLitePlanID,
			Name:        AzureLitePlanName,
			Description: "Azure Lite",
			Metadata: &domain.ServicePlanMetadata{
				DisplayName: "Azure Lite",
			},
			Schemas: &domain.ServiceSchemas{
				Instance: domain.ServiceInstanceSchema{
					Create: domain.Schema{
						Parameters: make(map[string]interface{}),
					},
				},
			},
		},
		provisioningRawSchema: AzureSchema([]string{"Standard_D4_v3"}),
	},
	TrialPlanID: {
		PlanDefinition: domain.ServicePlan{
			ID:          TrialPlanID,
			Name:        TrialPlanName,
			Description: "Trial",
			Metadata: &domain.ServicePlanMetadata{
				DisplayName: "Trial",
			},
			Schemas: &domain.ServiceSchemas{
				Instance: domain.ServiceInstanceSchema{
					Create: domain.Schema{
						Parameters: make(map[string]interface{}),
					},
				},
			},
		},
		provisioningRawSchema: TrialSchema(),
	},
}

func IsTrialPlan(planID string) bool {
	switch planID {
	case TrialPlanID:
		return true
	default:
		return false
	}
}
