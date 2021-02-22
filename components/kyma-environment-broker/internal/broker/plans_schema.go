package broker

type RootSchema struct {
	Schema string `json:"$schema"`
	Type
	Properties interface{} `json:"properties"`
	Required   []string    `json:"required"`

	// Specified to true enables form view on website
	ShowFormView bool `json:"_show_form_view"`
	// Specifies in what order properties will be displayed on the form
	ControlsOrder []string `json:"_controlsOrder"`
}

type ProvisioningProperties struct {
	Name          Type  `json:"name"`
	Region        *Type `json:"region,omitempty"`
	MachineType   *Type `json:"machineType,omitempty"`
	AutoScalerMin *Type `json:"autoScalerMin,omitempty"`
	AutoScalerMax *Type `json:"autoScalerMax,omitempty"`
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

func NameProperty() Type {
	return Type{
		Type:      "string",
		Title:     "Cluster Name",
		MinLength: 1,
	}
}

// NewProvisioningProperties creates a new properties for different plans
// Note that the order of properties will be the same in the form on the website
func NewProvisioningProperties(machineTypes []string, regions []string) ProvisioningProperties {
	return ProvisioningProperties{
		Name: NameProperty(),
		Region: &Type{
			Type: "string",
			Enum: ToInterfaceSlice(regions),
		},
		MachineType: &Type{
			Type: "string",
			Enum: ToInterfaceSlice(machineTypes),
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

func NewSchema(properties ProvisioningProperties, controlsOrder []string) RootSchema {
	return RootSchema{
		Schema: "http://json-schema.org/draft-04/schema#",
		Type: Type{
			Type: "object",
		},
		Properties:    properties,
		ShowFormView:  true,
		Required:      []string{"name"},
		ControlsOrder: controlsOrder,
	}
}

func DefaultControlsOrder() []string {
	return []string{"name", "region", "machineType", "autoScalerMin", "autoScalerMax"}
}

func ToInterfaceSlice(input []string) []interface{} {
	interfaces := make([]interface{}, len(input))
	for i, item := range input {
		interfaces[i] = item
	}
	return interfaces
}
