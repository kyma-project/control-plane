package internal

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

const (
	LicenceTypeLite      = "TestDevelopmentAndDemo"
	oidcValidSigningAlgs = "RS256,RS384,RS512,ES256,ES384,ES512,PS256,PS384,PS512"
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

func (o *OIDCConfigDTO) Validate() error {
	errs := make([]string, 0)
	if len(o.ClientID) == 0 {
		errs = append(errs, "clientID must not be empty")
	}
	if len(o.IssuerURL) == 0 {
		errs = append(errs, "issuerURL must not be empty")
	} else {
		issuer, err := url.Parse(o.IssuerURL)
		if err != nil || (issuer != nil && len(issuer.Host) == 0) {
			errs = append(errs, "issuerURL must be a valid URL")
		}
		if issuer != nil && issuer.Scheme != "https" {
			errs = append(errs, "issuerURL must have https scheme")
		}
	}
	if len(o.SigningAlgs) != 0 {
		validSigningAlgs := o.validSigningAlgsSet()
		for _, providedAlg := range o.SigningAlgs {
			if !validSigningAlgs[providedAlg] {
				errs = append(errs, "signingAlgs must contain valid signing algorithm(s)")
				break
			}
		}
	}

	if len(errs) > 0 {
		err := fmt.Errorf(strings.Join(errs, ", "))
		return err
	}
	return nil
}

func (o *OIDCConfigDTO) validSigningAlgsSet() map[string]bool {
	algs := strings.Split(oidcValidSigningAlgs, ",")
	signingAlgsSet := make(map[string]bool, len(algs))

	for _, v := range algs {
		signingAlgsSet[v] = true
	}

	return signingAlgsSet
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

	p.Parameters.TargetSecret = nil
	p.Parameters.LicenceType = nil
	input.Parameters.LicenceType = nil

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
func (p AutoScalerParameters) Validate(planMin, planMax int) error {
	min, max := planMin, planMax
	if p.AutoScalerMin != nil {
		min = *p.AutoScalerMin
	}
	if p.AutoScalerMax != nil {
		max = *p.AutoScalerMax
	}
	if min > max {
		userMin := fmt.Sprintf("%v", p.AutoScalerMin)
		if p.AutoScalerMin != nil {
			userMin = fmt.Sprintf("%v", *p.AutoScalerMin)
		}
		userMax := fmt.Sprintf("%v", p.AutoScalerMax)
		if p.AutoScalerMax != nil {
			userMax = fmt.Sprintf("%v", *p.AutoScalerMax)
		}
		return fmt.Errorf("AutoScalerMax %v should be larger than AutoScalerMin %v. User provided values min:%v, max:%v; plan defaults min:%v, max:%v", max, min, userMin, userMax, planMin, planMax)
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

	// Expired - means that the trial SKR is marked as expired
	Expired bool `json:"expired"`
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
	TenantID              string                             `json:"tenant_id"`
	SubAccountID          string                             `json:"subaccount_id"`
	GlobalAccountID       string                             `json:"globalaccount_id"`
	SMOperatorCredentials *ServiceManagerOperatorCredentials `json:"sm_operator_credentials,omitempty"`
	Active                *bool                              `json:"active,omitempty"`
	UserID                string                             `json:"user_id"`
	IsMigration           bool                               `json:"isMigration"`
	CommercialModel       *string                            `json:"commercial_model,omitempty"`
	LicenseType           *string                            `json:"license_type,omitempty"`
	Origin                *string                            `json:"origin,omitempty"`
	Platform              *string                            `json:"platform,omitempty"`
	Region                *string                            `json:"region,omitempty"`
}

func UpdateERSContext(currentOperation, previousOperation ERSContext) ERSContext {
	if currentOperation.SMOperatorCredentials == nil {
		currentOperation.SMOperatorCredentials = previousOperation.SMOperatorCredentials
	}
	if currentOperation.CommercialModel == nil {
		currentOperation.CommercialModel = previousOperation.CommercialModel
	}
	if currentOperation.LicenseType == nil {
		currentOperation.LicenseType = previousOperation.LicenseType
	}
	if currentOperation.Origin == nil {
		currentOperation.Origin = previousOperation.Origin
	}
	if currentOperation.Platform == nil {
		currentOperation.Platform = previousOperation.Platform
	}
	if currentOperation.Region == nil {
		currentOperation.Region = previousOperation.Region
	}
	return currentOperation
}

func (e ERSContext) DisableEnterprisePolicyFilter() *bool {
	// the provisioner and gardener API expects the feature to be enabled by disablement flag
	// it feels counterintuitive but there is currently no plan in changing it, therefore
	// following code is written the way it's written
	disable := false
	if e.LicenseType == nil {
		return &disable
	}
	switch *e.LicenseType {
	case "CUSTOMER", "PARTNER", "TRIAL":
		disable = true
		return &disable
	}
	return &disable
}

func (e ERSContext) ERSUpdate() bool {
	if e.CommercialModel != nil {
		return true
	}
	if e.LicenseType != nil {
		return true
	}
	if e.Origin != nil {
		return true
	}
	if e.Platform != nil {
		return true
	}
	if e.Region != nil {
		return true
	}
	return false
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

type ServiceManagerOperatorCredentials struct {
	ClientID          string `json:"clientid"`
	ClientSecret      string `json:"clientsecret"`
	ServiceManagerURL string `json:"sm_url"`
	URL               string `json:"url"`
	XSAppName         string `json:"xsappname"`
}
