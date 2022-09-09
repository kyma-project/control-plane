package broker

import "encoding/json"

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
	UpdateProperties

	Name        NameType `json:"name"`
	Region      *Type    `json:"region,omitempty"`
	MachineType *Type    `json:"machineType,omitempty"`
}

type UpdateProperties struct {
	AutoScalerMin  *Type     `json:"autoScalerMin,omitempty"`
	AutoScalerMax  *Type     `json:"autoScalerMax,omitempty"`
	OIDC           *OIDCType `json:"oidc,omitempty"`
	Administrators *Type     `json:"administrators,omitempty"`
}

func (up *UpdateProperties) IncludeAdditional() {
	up.OIDC = NewOIDCSchema()
	up.Administrators = AdministratorsProperty()
}

type OIDCProperties struct {
	ClientID       Type `json:"clientID"`
	GroupsClaim    Type `json:"groupsClaim"`
	IssuerURL      Type `json:"issuerURL"`
	SigningAlgs    Type `json:"signingAlgs"`
	UsernameClaim  Type `json:"usernameClaim"`
	UsernamePrefix Type `json:"usernamePrefix"`
}

type OIDCType struct {
	Type
	Properties OIDCProperties `json:"properties"`
	Required   []string       `json:"required"`
}

type Type struct {
	Type        string `json:"type"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Minimum     int    `json:"minimum,omitempty"`
	Maximum     int    `json:"maximum,omitempty"`
	MinLength   int    `json:"minLength,omitempty"`
	MaxLength   int    `json:"maxLength,omitempty"`

	// Regex pattern to match against string type of fields.
	// If not specified for strings user can pass empty string with whitespaces only.
	Pattern         string            `json:"pattern,omitempty"`
	Default         interface{}       `json:"default,omitempty"`
	Example         interface{}       `json:"example,omitempty"`
	Enum            []interface{}     `json:"enum,omitempty"`
	EnumDisplayName map[string]string `json:"_enumDisplayName,omitempty"`
	Items           []Type            `json:"items,omitempty"`
	AdditionalItems *bool             `json:"additionalItems,omitempty"`
	UniqueItems     *bool             `json:"uniqueItems,omitempty"`
}

type NameType struct {
	Type
	BTPdefaultTemplate BTPdefaultTemplate `json:"_BTPdefaultTemplate,omitempty"`
}

type BTPdefaultTemplate struct {
	Elements  []string `json:"elements,omitempty"`
	Separator string   `json:"separator,omitempty"`
}

func NameProperty() NameType {
	return NameType{
		Type: Type{
			Type:  "string",
			Title: "Cluster Name",
			// Allows for all alphanumeric characters, '_', and '-'
			Pattern:   "^[a-zA-Z0-9-]*$",
			MinLength: 1,
		},
		BTPdefaultTemplate: BTPdefaultTemplate{
			Elements: []string{"saSubdomain"},
		},
	}
}

// NewProvisioningProperties creates a new properties for different plans
// Note that the order of properties will be the same in the form on the website
func NewProvisioningProperties(machineTypesDisplay map[string]string, machineTypes, regions []string, update bool) ProvisioningProperties {

	properties := ProvisioningProperties{
		UpdateProperties: UpdateProperties{
			AutoScalerMin: &Type{
				Type:        "integer",
				Minimum:     2,
				Default:     3,
				Description: "Specifies the minimum number of virtual machines to create",
			},
			AutoScalerMax: &Type{
				Type:        "integer",
				Minimum:     2,
				Maximum:     80,
				Default:     20,
				Description: "Specifies the maximum number of virtual machines to create",
			},
		},
		Name: NameProperty(),
		Region: &Type{
			Type: "string",
			Enum: ToInterfaceSlice(regions),
		},
		MachineType: &Type{
			Type:            "string",
			Enum:            ToInterfaceSlice(machineTypes),
			EnumDisplayName: machineTypesDisplay,
		},
	}

	if update {
		properties.AutoScalerMax.Default = nil
		properties.AutoScalerMin.Default = nil
	}

	return properties
}

func NewOIDCSchema() *OIDCType {
	return &OIDCType{
		Type: Type{Type: "object", Description: "OIDC configuration"},
		Properties: OIDCProperties{
			ClientID:       Type{Type: "string", Description: "The client ID for the OpenID Connect client."},
			IssuerURL:      Type{Type: "string", Description: "The URL of the OpenID issuer, only HTTPS scheme will be accepted."},
			GroupsClaim:    Type{Type: "string", Description: "If provided, the name of a custom OpenID Connect claim for specifying user groups."},
			UsernameClaim:  Type{Type: "string", Description: "The OpenID claim to use as the user name."},
			UsernamePrefix: Type{Type: "string", Description: "If provided, all usernames will be prefixed with this value. If not provided, username claims other than 'email' are prefixed by the issuer URL to avoid clashes. To skip any prefixing, provide the value '-'."},
			SigningAlgs: Type{
				Type: "array",
				Items: []Type{{
					Type: "string",
				}},
				Description: "List of allowed JOSE asymmetric signing algorithms.",
			},
		},
		Required: []string{"clientID", "issuerURL"},
	}
}

func NewSchema(properties interface{}, update bool) *RootSchema {
	schema := &RootSchema{
		Schema: "http://json-schema.org/draft-04/schema#",
		Type: Type{
			Type: "object",
		},
		Properties:   properties,
		ShowFormView: true,
		Required:     []string{"name"},
	}

	if update {
		schema.Required = []string{}
	}

	return schema
}

func unmarshalOrPanic(from, to interface{}) interface{} {
	if from != nil {
		marshaled := Marshal(from)
		err := json.Unmarshal(marshaled, to)
		if err != nil {
			panic(err)
		}
	}
	return to
}

func DefaultControlsOrder() []string {
	return []string{"name", "region", "machineType", "autoScalerMin", "autoScalerMax", "zonesCount", "oidc", "administrators"}
}

func ToInterfaceSlice(input []string) []interface{} {
	interfaces := make([]interface{}, len(input))
	for i, item := range input {
		interfaces[i] = item
	}
	return interfaces
}

func AdministratorsProperty() *Type {
	return &Type{
		Type:        "array",
		Title:       "Administrators",
		Description: "Specifies the list of runtime administrators",
		Items: []Type{{
			Type: "string",
		}},
	}
}
