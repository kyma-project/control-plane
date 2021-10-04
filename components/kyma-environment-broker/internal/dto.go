package internal

import (
	"fmt"
	"reflect"
)

const (
	LicenceTypeLite = "TestDevelopmentAndDemo"
)

type OIDCConfigDTO struct {
	ClientID       string   `json:"clientID" yaml:"clientID"`
	GroupsClaim    string   `json:"groupsClaim" yaml:"groupsClaim"`
	IssuerURL      string   `json:"issuerURL" yaml:"issuerURL"`
	SigningAlgs    []string `json:"signingAlgs" yaml:"signingAlgs"`
	UsernameClaim  string   `json:"usernameClaim" yaml:"usernameClaim"`
	UsernamePrefix string   `json:"usernamePrefix" yaml:"usernamePrefix"`
}

func (o *OIDCConfigDTO) IsProvided() bool {
	if o == nil {
		return false
	}
	if o.ClientID == "" && o.IssuerURL == "" && o.GroupsClaim == "" && o.UsernamePrefix == "" && o.UsernameClaim == "" && len(o.SigningAlgs) == 0 {
		return false
	}
	return true
}

type ProvisioningParameters struct {
	PlanID     string                    `json:"plan_id"`
	ServiceID  string                    `json:"service_id"`
	ErsContext ERSContext                `json:"ers_context"`
	Parameters ProvisioningParametersDTO `json:"parameters"`

	// PlatformRegion defines the Platform region send in the request path, terminology:
	//  - `Platform` is a place where KEB is registered and which later sends request to KEB.
	//  - `Region` value is use e.g. for billing integration such as EDP.
	PlatformRegion string `json:"platform_region"`

	PlatformProvider CloudProvider `json:"platform_provider"`
}

func (p ProvisioningParameters) IsEqual(input ProvisioningParameters) bool {
	if p.PlanID != input.PlanID {
		return false
	}
	if p.ServiceID != input.ServiceID {
		return false
	}
	if p.PlatformRegion != input.PlatformRegion {
		return false
	}

	if !reflect.DeepEqual(p.ErsContext, input.ErsContext) {
		return false
	}
	if !reflect.DeepEqual(p.Parameters, input.Parameters) {
		return false
	}

	return true
}

type CloudProvider string

const (
	Azure           CloudProvider = "Azure"
	AWS             CloudProvider = "AWS"
	GCP             CloudProvider = "GCP"
	UnknownProvider CloudProvider = "unknown"
	Openstack       CloudProvider = "OpenStack"
)

type AutoScalerParameters struct {
	AutoScalerMin  *int `json:"autoScalerMin"`
	AutoScalerMax  *int `json:"autoScalerMax"`
	MaxSurge       *int `json:"maxSurge"`
	MaxUnavailable *int `json:"maxUnavailable"`
}

// FIXME: this is a makeshift check until the provisioner is capable of returning error messages
// https://github.com/kyma-project/control-plane/issues/946
func (p AutoScalerParameters) Validate() error {
	if p.AutoScalerMin == nil && p.AutoScalerMax == nil {
		return nil
	}
	if p.AutoScalerMin == nil || p.AutoScalerMax == nil {
		missing := "Min"
		if p.AutoScalerMax == nil {
			missing = "Max"
		}
		return fmt.Errorf("When setting Auto Scaler Min/Max you need to provide both, missing: %v", missing)
	}
	if *p.AutoScalerMin > *p.AutoScalerMax {
		return fmt.Errorf("Auto Scaler Max %v should be larger than Auto Scaler Min %v", *p.AutoScalerMax, *p.AutoScalerMin)
	}
	return nil
}

type ProvisioningParametersDTO struct {
	AutoScalerParameters `json:",inline"`

	Name         string  `json:"name"`
	TargetSecret *string `json:"targetSecret"`
	VolumeSizeGb *int    `json:"volumeSizeGb"`
	MachineType  *string `json:"machineType"`
	Region       *string `json:"region"`
	Purpose      *string `json:"purpose"`
	// LicenceType - based on this parameter, some options can be enabled/disabled when preparing the input
	// for the provisioner e.g. use default overrides for SKR instead overrides from resource
	// with "provisioning-runtime-override" label when LicenceType is "TestDevelopmentAndDemo"
	LicenceType                 *string  `json:"licence_type"`
	Zones                       []string `json:"zones"`
	ZonesCount                  *int     `json:"zonesCount"`
	OptionalComponentsToInstall []string `json:"components"`
	KymaVersion                 string   `json:"kymaVersion"`
	OverridesVersion            string   `json:"overridesVersion"`
	RuntimeAdministrators       []string `json:"administrators"`
	//Provider - used in Trial plan to determine which cloud provider to use during provisioning
	Provider *CloudProvider `json:"provider"`

	OIDC *OIDCConfigDTO `json:"oidc,omitempty"`
}

type UpdatingParametersDTO struct {
	AutoScalerParameters `json:",inline"`

	OIDC                  *OIDCConfigDTO `json:"oidc,omitempty"`
	RuntimeAdministrators []string       `json:"administrators,omitempty"`
}

func (u UpdatingParametersDTO) UpdateAutoScaler(p *ProvisioningParametersDTO) bool {
	updated := false
	if u.AutoScalerMin != nil {
		updated = true
		p.AutoScalerMin = u.AutoScalerMin
	}
	if u.AutoScalerMax != nil {
		updated = true
		p.AutoScalerMax = u.AutoScalerMax
	}
	if u.MaxSurge != nil {
		updated = true
		p.MaxSurge = u.MaxSurge
	}
	if u.MaxUnavailable != nil {
		updated = true
		p.MaxUnavailable = u.MaxUnavailable
	}
	return updated
}

type ERSContext struct {
	TenantID        string                  `json:"tenant_id"`
	SubAccountID    string                  `json:"subaccount_id"`
	GlobalAccountID string                  `json:"globalaccount_id"`
	ServiceManager  *ServiceManagerEntryDTO `json:"sm_platform_credentials,omitempty"`
	Active          *bool                   `json:"active,omitempty"`
	UserID          string                  `json:"user_id"`
}

type ServiceManagerEntryDTO struct {
	Credentials ServiceManagerCredentials `json:"credentials"`
	URL         string                    `json:"url"`
}

type ServiceManagerCredentials struct {
	BasicAuth ServiceManagerBasicAuth `json:"basic"`
}

type ServiceManagerBasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
