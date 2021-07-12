package broker

import (
	"encoding/json"
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

func AWSRegions() []string {
	// be aware of zones defined in internal/provider/aws_provider.go
	return []string{"eu-central-1", "eu-west-2", "ca-central-1", "sa-east-1", "us-east-1", "us-west-1",
		"ap-northeast-1", "ap-northeast-2", "ap-south-1", "ap-southeast-1", "ap-southeast-2"}
}

func OpenStackRegions() []string {
	return []string{"eu-de-1", "ap-sa-1"}
}

func OpenStackSchema(machineTypes []string) []byte {
	properties := NewProvisioningProperties(machineTypes, OpenStackRegions())
	schema := NewSchema(properties, DefaultControlsOrder())

	bytes, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}
	return bytes
}

func GCPSchema(machineTypes []string) []byte {
	properties := NewProvisioningProperties(machineTypes, GCPRegions())
	schema := NewSchema(properties, DefaultControlsOrder())

	bytes, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}
	return bytes
}

func AWSSchema(machineTypes []string) []byte {
	properties := NewProvisioningProperties(machineTypes, AWSRegions())
	schema := NewSchema(properties, DefaultControlsOrder())

	bytes, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}
	return bytes
}

func AWSHASchema(machineTypes []string) []byte {
	properties := NewProvisioningProperties(machineTypes, AWSRegions())
	properties.ZonesCount = &Type{
		Type:        "integer",
		Minimum:     2,
		Maximum:     3,
		Default:     2,
		Description: "Specifies the number of availability zones for HA cluster",
	}
	awsHaControlsOrder := DefaultControlsOrder()
	awsHaControlsOrder = append(awsHaControlsOrder, "zonesCount")
	schema := NewSchema(properties, awsHaControlsOrder)

	properties.AutoScalerMin.Default = 4
	properties.AutoScalerMin.Minimum = 4

	properties.AutoScalerMax.Minimum = 4

	bytes, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}
	return bytes
}

func AzureSchema(machineTypes []string) []byte {
	properties := NewProvisioningProperties(machineTypes, AzureRegions())
	schema := NewSchema(properties, DefaultControlsOrder())

	bytes, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}
	return bytes
}

func AzureLiteSchema(machineTypes []string) []byte {
	properties := NewProvisioningProperties(machineTypes, AzureRegions())
	properties.AutoScalerMax.Maximum = 4
	properties.AutoScalerMax.Default = 2

	schema := NewSchema(properties, DefaultControlsOrder())

	bytes, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}
	return bytes
}

func FreemiumSchema(provider internal.CloudProvider) []byte {
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
			//OIDC: NewOIDCSchema(),
		}, []string{"name", "region"})

	bytes, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}
	return bytes
}

func AzureHASchema(machineTypes []string) []byte {
	properties := NewProvisioningProperties(machineTypes, AzureRegions())
	properties.ZonesCount = &Type{
		Type:        "integer",
		Minimum:     2,
		Maximum:     3,
		Default:     2,
		Description: "Specifies the number of availability zones for HA cluster",
	}
	azureHaControlsOrder := DefaultControlsOrder()
	azureHaControlsOrder = append(azureHaControlsOrder, "zonesCount")
	schema := NewSchema(properties, azureHaControlsOrder)

	properties.AutoScalerMin.Default = 4
	properties.AutoScalerMin.Minimum = 4

	properties.AutoScalerMax.Minimum = 4

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
			//OIDC: NewOIDCSchema(),
		}, []string{"name"})

	bytes, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}
	return bytes
}

type Plan struct {
	PlanDefinition        domain.ServicePlan
	provisioningRawSchema []byte
}

// plans is designed to hold plan defaulting logic
// keep internal/hyperscaler/azure/config.go in sync with any changes to available zones
func Plans(plans PlansConfig, provider internal.CloudProvider) map[string]Plan {
	return map[string]Plan{
		AWSPlanID: {
			PlanDefinition: domain.ServicePlan{
				ID:          AWSPlanID,
				Name:        AWSPlanName,
				Description: defaultDescription(AWSPlanName, plans),
				Metadata:    defaultMetadata(AWSPlanName, plans), Schemas: &domain.ServiceSchemas{
					Instance: domain.ServiceInstanceSchema{
						Create: domain.Schema{
							Parameters: make(map[string]interface{}),
						},
					},
				},
			},
			provisioningRawSchema: AWSSchema([]string{"m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge"}),
		},
		AWSHAPlanID: {
			PlanDefinition: domain.ServicePlan{
				ID:          AWSHAPlanID,
				Name:        AWSHAPlanName,
				Description: defaultDescription(AWSHAPlanName, plans),
				Metadata:    defaultMetadata(AWSHAPlanName, plans), Schemas: &domain.ServiceSchemas{
					Instance: domain.ServiceInstanceSchema{
						Create: domain.Schema{
							Parameters: make(map[string]interface{}),
						},
					},
				},
			},
			provisioningRawSchema: AWSHASchema([]string{"m5d.xlarge"}),
		},
		GCPPlanID: {
			PlanDefinition: domain.ServicePlan{
				ID:          GCPPlanID,
				Name:        GCPPlanName,
				Description: defaultDescription(GCPPlanName, plans),
				Metadata:    defaultMetadata(GCPPlanName, plans),
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
		OpenStackPlanID: {
			PlanDefinition: domain.ServicePlan{
				ID:          OpenStackPlanID,
				Name:        OpenStackPlanName,
				Description: defaultDescription(OpenStackPlanName, plans),
				Metadata:    defaultMetadata(OpenStackPlanName, plans),
				Schemas: &domain.ServiceSchemas{
					Instance: domain.ServiceInstanceSchema{
						Create: domain.Schema{
							Parameters: make(map[string]interface{}),
						},
					},
				},
			},
			provisioningRawSchema: OpenStackSchema([]string{"m2.xlarge", "m1.2xlarge"}),
		},
		AzurePlanID: {
			PlanDefinition: domain.ServicePlan{
				ID:          AzurePlanID,
				Name:        AzurePlanName,
				Description: defaultDescription(AzurePlanName, plans),
				Metadata:    defaultMetadata(AzurePlanName, plans),
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
				Description: defaultDescription(AzureLitePlanName, plans),
				Metadata:    defaultMetadata(AzureLitePlanName, plans),
				Schemas: &domain.ServiceSchemas{
					Instance: domain.ServiceInstanceSchema{
						Create: domain.Schema{
							Parameters: make(map[string]interface{}),
						},
					},
				},
			},
			provisioningRawSchema: AzureLiteSchema([]string{"Standard_D4_v3"}),
		},
		FreemiumPlanID: {
			PlanDefinition: domain.ServicePlan{
				ID:          FreemiumPlanID,
				Name:        FreemiumPlanName,
				Description: defaultDescription(FreemiumPlanName, plans),
				Metadata:    defaultMetadata(FreemiumPlanName, plans),
				Schemas: &domain.ServiceSchemas{
					Instance: domain.ServiceInstanceSchema{
						Create: domain.Schema{
							Parameters: make(map[string]interface{}),
						},
					},
				},
			},
			provisioningRawSchema: FreemiumSchema(provider),
		},
		AzureHAPlanID: {
			PlanDefinition: domain.ServicePlan{
				ID:          AzureHAPlanID,
				Name:        AzureHAPlanName,
				Description: defaultDescription(AzureHAPlanName, plans),
				Metadata:    defaultMetadata(AzureHAPlanName, plans),
				Schemas: &domain.ServiceSchemas{
					Instance: domain.ServiceInstanceSchema{
						Create: domain.Schema{
							Parameters: make(map[string]interface{}),
						},
					},
				},
			},
			provisioningRawSchema: AzureHASchema([]string{"Standard_D4_v3"}),
		},
		TrialPlanID: {
			PlanDefinition: domain.ServicePlan{
				ID:          TrialPlanID,
				Name:        TrialPlanName,
				Description: defaultDescription(TrialPlanName, plans),
				Metadata:    defaultMetadata(TrialPlanName, plans),
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
