package model

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-version"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model/infrastructure/aws"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model/infrastructure/azure"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryRuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const (
	SubAccountLabel = "subaccount"
	AccountLabel    = "account"

	LicenceTypeAnnotation                = "kcp.provisioner.kyma-project.io/licence-type"
	EuAccessAnnotation                   = "support.gardener.cloud/eu-access-for-cluster-nodes"
	ShootNetworkingFilterExtensionType   = "shoot-networking-filter"
	ShootNetworkingFilterDisabledDefault = true
)

var networkingType = "calico"

type OIDCConfig struct {
	ClientID       string   `json:"clientID"`
	GroupsClaim    string   `json:"groupsClaim"`
	IssuerURL      string   `json:"issuerURL"`
	SigningAlgs    []string `json:"signingAlgs"`
	UsernameClaim  string   `json:"usernameClaim"`
	UsernamePrefix string   `json:"usernamePrefix"`
}

type DNSConfig struct {
	Domain    string         `json:"domain"`
	Providers []*DNSProvider `json:"providers"`
}

type DNSProvider struct {
	DomainsInclude []string `json:"domainsInclude" db:"-"`
	Primary        bool     `json:"primary" db:"is_primary"`
	SecretName     string   `json:"secretName" db:"secret_name"`
	Type           string   `json:"type" db:"type"`
}

type GardenerConfig struct {
	AutoScalerMax                       int
	AutoScalerMin                       int
	ClusterID                           string
	ControlPlaneFailureTolerance        *string
	DNSConfig                           *DNSConfig
	DiskType                            *string
	EnableKubernetesVersionAutoUpdate   bool
	EnableMachineImageVersionAutoUpdate bool
	EuAccess                            bool
	ExposureClassName                   *string
	GardenerProviderConfig              GardenerProviderConfig
	ID                                  string
	KubernetesVersion                   string
	LicenceType                         *string
	MachineImage                        *string
	MachineImageVersion                 *string
	MachineType                         string
	MaxSurge                            int
	MaxUnavailable                      int
	Name                                string
	OIDCConfig                          *OIDCConfig
	PodsCIDR                            *string
	ProjectName                         string
	Provider                            string
	Purpose                             *string
	Region                              string
	Seed                                string
	ServicesCIDR                        *string
	ShootNetworkingFilterDisabled       *bool
	TargetSecret                        string
	VolumeSizeGB                        *int
	WorkerCidr                          string
}

type ExtensionProviderConfig struct {
	// ApiVersion is gardener extension api version
	ApiVersion string `json:"apiVersion"`
	// DnsProviderReplication indicates whether dnsProvider replication is on
	DNSProviderReplication *DNSProviderReplication `json:"dnsProviderReplication,omitempty"`
	// ShootIssuers indicates whether shoot Issuers are on
	ShootIssuers *ShootIssuers `json:"shootIssuers,omitempty"`
	// Kind is extension type
	Kind string `json:"kind"`
}

type DNSProviderReplication struct {
	// Enabled indicates whether replication is on
	Enabled bool `json:"enabled"`
}

type ShootIssuers struct {
	// Enabled indicates whether shoot Issuers are on
	Enabled bool `json:"enabled"`
}

func NewDNSConfig() *ExtensionProviderConfig {
	return &ExtensionProviderConfig{
		ApiVersion:             "service.dns.extensions.gardener.cloud/v1alpha1",
		DNSProviderReplication: &DNSProviderReplication{Enabled: true},
		Kind:                   "DNSConfig",
	}
}

func NewCertConfig() *ExtensionProviderConfig {
	return &ExtensionProviderConfig{
		ApiVersion:   "service.cert.extensions.gardener.cloud/v1alpha1",
		ShootIssuers: &ShootIssuers{Enabled: true},
		Kind:         "CertConfig",
	}
}

func (c GardenerConfig) ToShootTemplate(namespace string, accountId string, subAccountId string, oidcConfig *OIDCConfig, dnsInputConfig *DNSConfig) (*gardener_types.Shoot, apperrors.AppError) {

	var seed *string = nil
	if c.Seed != "" {
		seed = util.PtrTo(c.Seed)
	}
	var purpose *gardener_types.ShootPurpose = nil
	if util.NotNilOrEmpty(c.Purpose) {
		p := gardener_types.ShootPurpose(*c.Purpose)
		purpose = &p
	}

	var exposureClassName *string = nil

	if util.NotNilOrEmpty(c.ExposureClassName) {
		exposureClassName = c.ExposureClassName
	}

	annotations := make(map[string]string)
	if c.LicenceType != nil {
		annotations[LicenceTypeAnnotation] = *c.LicenceType
	}
	if c.EuAccess {
		annotations[EuAccessAnnotation] = fmt.Sprintf("%t", c.EuAccess)
	}

	dnsConfig := NewDNSConfig()
	jsonDNSConfig, encodingErr := json.Marshal(dnsConfig)
	if encodingErr != nil {
		return nil, apperrors.Internal("error encoding DNS extension config: %s", encodingErr.Error())
	}

	certConfig := NewCertConfig()
	jsonCertConfig, encodingErr := json.Marshal(certConfig)
	if encodingErr != nil {
		return nil, apperrors.Internal("error encoding Cert extension config: %s", encodingErr.Error())
	}

	var controlPlane *gardener_types.ControlPlane = nil
	if c.ControlPlaneFailureTolerance != nil && *c.ControlPlaneFailureTolerance != "" {
		controlPlane = &gardener_types.ControlPlane{
			HighAvailability: &gardener_types.HighAvailability{
				FailureTolerance: gardener_types.FailureTolerance{
					Type: gardener_types.FailureToleranceType(*c.ControlPlaneFailureTolerance),
				},
			},
		}
	}
	shoot := &gardener_types.Shoot{
		ObjectMeta: v1.ObjectMeta{
			Name:      c.Name,
			Namespace: namespace,
			Labels: map[string]string{
				SubAccountLabel: subAccountId,
				AccountLabel:    accountId,
			},
			Annotations: annotations,
		},
		Spec: gardener_types.ShootSpec{
			SecretBindingName: &c.TargetSecret,
			SeedName:          seed,
			Region:            c.Region,
			Kubernetes: gardener_types.Kubernetes{
				Version: c.KubernetesVersion,
				KubeAPIServer: &gardener_types.KubeAPIServerConfig{
					OIDCConfig: gardenerOidcConfig(oidcConfig),
				},
				EnableStaticTokenKubeconfig: util.PtrTo(false),
			},
			Networking: &gardener_types.Networking{
				Type:     &networkingType, // Default value - we may consider adding it to API (if Hydroform will support it)
				Nodes:    util.PtrTo(c.GardenerProviderConfig.NodeCIDR(c)),
				Pods:     c.PodsCIDR,
				Services: c.ServicesCIDR,
			},
			Purpose:           purpose,
			ExposureClassName: exposureClassName,
			Maintenance: &gardener_types.Maintenance{
				AutoUpdate: &gardener_types.MaintenanceAutoUpdate{
					KubernetesVersion:   c.EnableKubernetesVersionAutoUpdate,
					MachineImageVersion: &c.EnableMachineImageVersionAutoUpdate,
				},
			},
			DNS: gardenerDnsConfig(dnsInputConfig),
			Extensions: []gardener_types.Extension{
				{Type: "shoot-dns-service", ProviderConfig: &apimachineryRuntime.RawExtension{Raw: jsonDNSConfig}},
				{Type: "shoot-cert-service", ProviderConfig: &apimachineryRuntime.RawExtension{Raw: jsonCertConfig}},
				{Type: ShootNetworkingFilterExtensionType, Disabled: util.OkOrDefault(c.ShootNetworkingFilterDisabled, util.PtrTo(ShootNetworkingFilterDisabledDefault))},
			},
			ControlPlane: controlPlane,
		},
	}

	err := c.GardenerProviderConfig.ExtendShootConfig(c, shoot)
	if err != nil {
		return nil, err.Append("error extending shoot config with Provider")
	}

	return shoot, nil
}

func gardenerOidcConfig(oidcConfig *OIDCConfig) *gardener_types.OIDCConfig {
	if oidcConfig != nil {
		return &gardener_types.OIDCConfig{
			ClientID:       &oidcConfig.ClientID,
			GroupsClaim:    &oidcConfig.GroupsClaim,
			IssuerURL:      &oidcConfig.IssuerURL,
			SigningAlgs:    oidcConfig.SigningAlgs,
			UsernameClaim:  &oidcConfig.UsernameClaim,
			UsernamePrefix: &oidcConfig.UsernamePrefix,
		}
	}
	return nil
}

func gardenerDnsConfig(dnsConfig *DNSConfig) *gardener_types.DNS {
	dns := gardener_types.DNS{}

	if dnsConfig != nil {
		dns.Domain = &dnsConfig.Domain
		if len(dnsConfig.Providers) != 0 {
			for _, v := range dnsConfig.Providers {
				domainsInclude := &gardener_types.DNSIncludeExclude{
					Include: v.DomainsInclude,
				}

				dns.Providers = append(dns.Providers, gardener_types.DNSProvider{
					Domains:    domainsInclude,
					Primary:    &v.Primary,
					SecretName: &v.SecretName,
					Type:       &v.Type,
				})
			}

		}

		return &dns
	}

	return nil
}

type ProviderSpecificConfig string

func (c ProviderSpecificConfig) RawJSON() string {
	return string(c)
}

type GardenerProviderConfig interface {
	RawJSON() string
	NodeCIDR(gardenerConfig GardenerConfig) string
	AsProviderSpecificConfig() gqlschema.ProviderSpecificConfig
	ExtendShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError
	EditShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError
	ValidateShootConfigChange(shoot *gardener_types.Shoot) apperrors.AppError
}

func NewGardenerProviderConfigFromJSON(jsonData string) (GardenerProviderConfig, apperrors.AppError) { //TODO: change to detect Provider correctly
	var gcpProviderConfig gqlschema.GCPProviderConfigInput
	err := util.DecodeJson(jsonData, &gcpProviderConfig)
	if err == nil {
		return &GCPGardenerConfig{input: &gcpProviderConfig, ProviderSpecificConfig: ProviderSpecificConfig(jsonData)}, nil
	}

	var azureProviderConfig gqlschema.AzureProviderConfigInput
	err = util.DecodeJson(jsonData, &azureProviderConfig)
	if err == nil {
		return &AzureGardenerConfig{input: &azureProviderConfig, ProviderSpecificConfig: ProviderSpecificConfig(jsonData)}, nil
	}

	var awsProviderConfig gqlschema.AWSProviderConfigInput
	err = util.DecodeJson(jsonData, &awsProviderConfig)
	if err == nil {
		return &AWSGardenerConfig{input: &awsProviderConfig, ProviderSpecificConfig: ProviderSpecificConfig(jsonData)}, nil
	}

	var openStackProviderConfig gqlschema.OpenStackProviderConfigInput
	err = util.DecodeJson(jsonData, &openStackProviderConfig)
	if err == nil {
		return &OpenStackGardenerConfig{input: &openStackProviderConfig, ProviderSpecificConfig: ProviderSpecificConfig(jsonData)}, nil
	}

	return nil, apperrors.BadRequest("json data does not match any of Gardener providers")
}

type GCPGardenerConfig struct {
	ProviderSpecificConfig
	input *gqlschema.GCPProviderConfigInput `db:"-"`
}

func NewGCPGardenerConfig(input *gqlschema.GCPProviderConfigInput) (*GCPGardenerConfig, apperrors.AppError) {
	config, err := json.Marshal(input)
	if err != nil {
		return &GCPGardenerConfig{}, apperrors.Internal("failed to marshal GCP Gardener config")
	}

	return &GCPGardenerConfig{
		ProviderSpecificConfig: ProviderSpecificConfig(config),
		input:                  input,
	}, nil
}

func (c GCPGardenerConfig) NodeCIDR(gardenerConfig GardenerConfig) string {
	return gardenerConfig.WorkerCidr
}

func (c GCPGardenerConfig) AsProviderSpecificConfig() gqlschema.ProviderSpecificConfig {
	return gqlschema.GCPProviderConfig{Zones: c.input.Zones}
}

func (c GCPGardenerConfig) EditShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError {
	return updateShootConfig(gardenerConfig, shoot)
}

func (c GCPGardenerConfig) ValidateShootConfigChange(*gardener_types.Shoot) apperrors.AppError {
	return nil
}

func (c GCPGardenerConfig) ExtendShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError {
	shoot.Spec.CloudProfileName = "gcp"

	workers := []gardener_types.Worker{getWorkerConfig(gardenerConfig, c.input.Zones)}

	gcpInfra := NewGCPInfrastructure(gardenerConfig.WorkerCidr)
	jsonData, err := json.Marshal(gcpInfra)
	if err != nil {
		return apperrors.Internal("error encoding infrastructure config: %s", err.Error())
	}

	gcpControlPlane := NewGCPControlPlane(c.input.Zones)
	jsonCPData, err := json.Marshal(gcpControlPlane)
	if err != nil {
		return apperrors.Internal("error encoding control plane config: %s", err.Error())
	}

	shoot.Spec.Provider = gardener_types.Provider{

		Type:                 "gcp",
		ControlPlaneConfig:   &apimachineryRuntime.RawExtension{Raw: jsonCPData},
		InfrastructureConfig: &apimachineryRuntime.RawExtension{Raw: jsonData},
		Workers:              workers,
	}

	return nil
}

type AzureGardenerConfig struct {
	ProviderSpecificConfig
	input *gqlschema.AzureProviderConfigInput `db:"-"`
}

func NewAzureGardenerConfig(input *gqlschema.AzureProviderConfigInput) (*AzureGardenerConfig, apperrors.AppError) {
	config, err := json.Marshal(input)
	if err != nil {
		return &AzureGardenerConfig{}, apperrors.Internal("failed to marshal Azure Gardener config")
	}

	return &AzureGardenerConfig{
		ProviderSpecificConfig: ProviderSpecificConfig(config),
		input:                  input,
	}, nil
}

func (c AzureGardenerConfig) NodeCIDR(GardenerConfig) string {
	return c.input.VnetCidr
}

func (c AzureGardenerConfig) AsProviderSpecificConfig() gqlschema.ProviderSpecificConfig {
	var zones []*gqlschema.AzureZone = nil
	if len(c.input.AzureZones) > 0 {
		zones = make([]*gqlschema.AzureZone, 0)
	}

	for _, inputZone := range c.input.AzureZones {
		zone := &gqlschema.AzureZone{
			Name: inputZone.Name,
			Cidr: inputZone.Cidr,
		}
		zones = append(zones, zone)
	}

	return gqlschema.AzureProviderConfig{
		VnetCidr:                     &c.input.VnetCidr,
		Zones:                        c.input.Zones,
		AzureZones:                   zones,
		EnableNatGateway:             c.input.EnableNatGateway,
		IdleConnectionTimeoutMinutes: c.input.IdleConnectionTimeoutMinutes,
	}
}

type AWSGardenerConfig struct {
	ProviderSpecificConfig
	input *gqlschema.AWSProviderConfigInput `db:"-"`
}

func (c AzureGardenerConfig) ValidateShootConfigChange(shoot *gardener_types.Shoot) apperrors.AppError {
	// Check if the zone is already configured. Deny change to CIDR. Deny new zones (no support for extension of zones).
	infra := azure.InfrastructureConfig{}
	if c.input.AzureZones != nil {
		err := json.Unmarshal(shoot.Spec.Provider.InfrastructureConfig.Raw, &infra)
		if err != nil {
			return apperrors.Internal("error decoding infrastructure config: %s", err.Error())
		}
	}
	for _, inputZone := range c.input.AzureZones {
		zoneFound := false
		for _, zone := range infra.Networks.Zones {
			if inputZone.Name == zone.Name {
				zoneFound = true
				if inputZone.Cidr != zone.CIDR {
					return apperrors.BadRequest("cannot change shoot network zone CIDR from %s to %s", zone.CIDR, inputZone.Cidr)
				}
			}
		}
		if !zoneFound {
			return apperrors.BadRequest("extension of shoot network zones is not supported")
		}
	}

	return nil
}

func (c AzureGardenerConfig) EditShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError {
	err := updateShootConfig(gardenerConfig, shoot)
	if err != nil {
		return err
	}
	if c.input.EnableNatGateway != nil {
		infra := azure.InfrastructureConfig{}
		err := json.Unmarshal(shoot.Spec.Provider.InfrastructureConfig.Raw, &infra)
		if err != nil {
			return apperrors.Internal("error decoding infrastructure config: %s", err.Error())
		}

		if len(c.input.AzureZones) == 0 {
			if *c.input.EnableNatGateway {
				if infra.Networks.NatGateway == nil {
					infra.Networks.NatGateway = &azure.NatGateway{}
				}
				infra.Networks.NatGateway.Enabled = *c.input.EnableNatGateway
				infra.Networks.NatGateway.IdleConnectionTimeoutMinutes = util.UnwrapOrDefault(c.input.IdleConnectionTimeoutMinutes, defaultConnectionTimeOutMinutes)
			} else {
				infra.Networks.NatGateway = nil
			}
		} else {
			for i := range infra.Networks.Zones {
				zone := infra.Networks.Zones[i]
				if *c.input.EnableNatGateway {
					if zone.NatGateway == nil {
						zone.NatGateway = &azure.NatGateway{}
					}
					zone.NatGateway.Enabled = *c.input.EnableNatGateway
					zone.NatGateway.IdleConnectionTimeoutMinutes = util.UnwrapOrDefault(c.input.IdleConnectionTimeoutMinutes, defaultConnectionTimeOutMinutes)
				} else {
					zone.NatGateway = nil
				}
				infra.Networks.Zones[i] = zone
			}
		}
		infra.Networks.VNet.CIDR = util.PtrTo(c.input.VnetCidr)
		jsonData, err := json.Marshal(infra)
		if err != nil {
			return apperrors.Internal("error encoding infrastructure config: %s", err.Error())
		}
		shoot.Spec.Provider.InfrastructureConfig = &apimachineryRuntime.RawExtension{Raw: jsonData}
	}
	return nil
}

func (c AzureGardenerConfig) ExtendShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError {
	shoot.Spec.CloudProfileName = "az"

	zoneNames := c.input.Zones
	if len(c.input.AzureZones) > 0 {
		zoneNames = getAzureZonesNames(c.input.AzureZones)
	}
	workers := []gardener_types.Worker{getWorkerConfig(gardenerConfig, zoneNames)}

	azInfra := NewAzureInfrastructure(gardenerConfig.WorkerCidr, c)
	jsonData, err := json.Marshal(azInfra)
	if err != nil {
		return apperrors.Internal("error encoding infrastructure config: %s", err.Error())
	}

	azureControlPlane := NewAzureControlPlane(zoneNames)
	jsonCPData, err := json.Marshal(azureControlPlane)
	if err != nil {
		return apperrors.Internal("error encoding control plane config: %s", err.Error())
	}

	shoot.Spec.Provider = gardener_types.Provider{
		Type:                 "azure",
		ControlPlaneConfig:   &apimachineryRuntime.RawExtension{Raw: jsonCPData},
		InfrastructureConfig: &apimachineryRuntime.RawExtension{Raw: jsonData},
		Workers:              workers,
	}

	return nil
}

func NewAWSGardenerConfig(input *gqlschema.AWSProviderConfigInput) (*AWSGardenerConfig, apperrors.AppError) {
	config, err := json.Marshal(input)
	if err != nil {
		return &AWSGardenerConfig{}, apperrors.Internal("failed to marshal AWS Gardener config")
	}

	return &AWSGardenerConfig{
		ProviderSpecificConfig: ProviderSpecificConfig(config),
		input:                  input,
	}, nil
}

func (c AWSGardenerConfig) NodeCIDR(GardenerConfig) string {
	return c.input.VpcCidr
}

func (c AWSGardenerConfig) AsProviderSpecificConfig() gqlschema.ProviderSpecificConfig {
	zones := make([]*gqlschema.AWSZone, 0)

	for _, inputZone := range c.input.AwsZones {
		zone := &gqlschema.AWSZone{
			Name:         &inputZone.Name,
			PublicCidr:   &inputZone.PublicCidr,
			InternalCidr: &inputZone.InternalCidr,
			WorkerCidr:   &inputZone.WorkerCidr,
		}
		zones = append(zones, zone)
	}

	return gqlschema.AWSProviderConfig{
		AwsZones: zones,
		VpcCidr:  &c.input.VpcCidr,
	}
}

func (c AWSGardenerConfig) ValidateShootConfigChange(shoot *gardener_types.Shoot) apperrors.AppError {
	infra := aws.InfrastructureConfig{}
	err := json.Unmarshal(shoot.Spec.Provider.InfrastructureConfig.Raw, &infra)
	if err != nil {
		return apperrors.Internal("error decoding infrastructure config: %s", err.Error())
	}
	for _, inputZone := range c.input.AwsZones {
		zoneFound := false
		for _, zone := range infra.Networks.Zones {
			if inputZone.Name == zone.Name {
				zoneFound = true
				if inputZone.WorkerCidr != zone.Workers {
					return apperrors.BadRequest("cannot change shoot network zone workers CIDR from %s to %s", zone.Workers, inputZone.WorkerCidr)
				}
				if inputZone.InternalCidr != zone.Internal {
					return apperrors.BadRequest("cannot change shoot network zone internal CIDR from %s to %s", zone.Internal, inputZone.InternalCidr)
				}
				if inputZone.PublicCidr != zone.Public {
					return apperrors.BadRequest("cannot change shoot network zone internal CIDR from %s to %s", zone.Public, inputZone.PublicCidr)
				}
			}
		}

		if !zoneFound {
			return apperrors.BadRequest("extension of shoot network zones is not supported")
		}
	}

	return nil
}

func (c AWSGardenerConfig) EditShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError {
	return updateShootConfig(gardenerConfig, shoot)
}

func (c AWSGardenerConfig) ExtendShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError {
	shoot.Spec.CloudProfileName = "aws"

	zoneNames := getAWSZonesNames(c.input.AwsZones)

	workers := []gardener_types.Worker{getWorkerConfig(gardenerConfig, zoneNames)}

	awsInfra := NewAWSInfrastructure(c)
	jsonData, err := json.Marshal(awsInfra)
	if err != nil {
		return apperrors.Internal("error encoding infrastructure config: %s", err.Error())
	}

	awsControlPlane := NewAWSControlPlane()
	jsonCPData, err := json.Marshal(awsControlPlane)
	if err != nil {
		return apperrors.Internal("error encoding control plane config: %s", err.Error())
	}

	shoot.Spec.Provider = gardener_types.Provider{
		Type:                 "aws",
		ControlPlaneConfig:   &apimachineryRuntime.RawExtension{Raw: jsonCPData},
		InfrastructureConfig: &apimachineryRuntime.RawExtension{Raw: jsonData},
		Workers:              workers,
	}

	return nil
}

type OpenStackGardenerConfig struct {
	ProviderSpecificConfig
	input *gqlschema.OpenStackProviderConfigInput `db:"-"`
}

func NewOpenStackGardenerConfig(input *gqlschema.OpenStackProviderConfigInput) (*OpenStackGardenerConfig, apperrors.AppError) {
	config, err := json.Marshal(input)
	if err != nil {
		return &OpenStackGardenerConfig{}, apperrors.Internal("failed to marshal OpenStack Gardener config")
	}

	return &OpenStackGardenerConfig{
		ProviderSpecificConfig: ProviderSpecificConfig(config),
		input:                  input,
	}, nil
}

func (c OpenStackGardenerConfig) NodeCIDR(gardenerConfig GardenerConfig) string {
	return gardenerConfig.WorkerCidr
}

func (c OpenStackGardenerConfig) AsProviderSpecificConfig() gqlschema.ProviderSpecificConfig {
	return gqlschema.OpenStackProviderConfig{
		Zones:                c.input.Zones,
		FloatingPoolName:     util.UnwrapOrZero(c.input.FloatingPoolName),
		CloudProfileName:     util.UnwrapOrZero(c.input.CloudProfileName),
		LoadBalancerProvider: c.input.LoadBalancerProvider,
	}
}

func (c OpenStackGardenerConfig) ValidateShootConfigChange(*gardener_types.Shoot) apperrors.AppError {
	return nil
}

func (c OpenStackGardenerConfig) EditShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError {
	return updateShootConfig(gardenerConfig, shoot)
}

func (c OpenStackGardenerConfig) ExtendShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError {
	shoot.Spec.CloudProfileName = util.UnwrapOrZero(c.input.CloudProfileName)

	workers := []gardener_types.Worker{getWorkerConfig(gardenerConfig, c.input.Zones)}

	openStackInfra := NewOpenStackInfrastructure(util.UnwrapOrZero(c.input.FloatingPoolName), gardenerConfig.WorkerCidr)

	jsonData, err := json.Marshal(openStackInfra)
	if err != nil {
		return apperrors.Internal("error encoding infrastructure config: %s", err.Error())
	}

	openstackControlPlane := NewOpenStackControlPlane(c.input.LoadBalancerProvider)
	jsonCPData, err := json.Marshal(openstackControlPlane)
	if err != nil {
		return apperrors.Internal("error encoding control plane config: %s", err.Error())
	}

	shoot.Spec.Provider = gardener_types.Provider{
		Type:                 "openstack",
		ControlPlaneConfig:   &apimachineryRuntime.RawExtension{Raw: jsonCPData},
		InfrastructureConfig: &apimachineryRuntime.RawExtension{Raw: jsonData},
		Workers:              workers,
	}

	return nil
}

func getWorkerConfig(gardenerConfig GardenerConfig, zones []string) gardener_types.Worker {
	worker := gardener_types.Worker{
		Name:           "cpu-worker-0",
		MaxSurge:       util.PtrTo(intstr.FromInt(gardenerConfig.MaxSurge)),
		MaxUnavailable: util.PtrTo(intstr.FromInt(gardenerConfig.MaxUnavailable)),
		Machine:        getMachineConfig(gardenerConfig),
		Maximum:        int32(gardenerConfig.AutoScalerMax),
		Minimum:        int32(gardenerConfig.AutoScalerMin),
		Zones:          zones,
	}

	if gardenerConfig.DiskType != nil && gardenerConfig.VolumeSizeGB != nil {
		worker.Volume = &gardener_types.Volume{
			Type:       gardenerConfig.DiskType,
			VolumeSize: fmt.Sprintf("%dGi", *gardenerConfig.VolumeSizeGB),
		}
	}

	return worker
}

func updateShootConfig(upgradeConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError {

	if upgradeConfig.KubernetesVersion != "" {
		shoot.Spec.Kubernetes.Version = upgradeConfig.KubernetesVersion

		adjustStaticKubeconfigFlag(upgradeConfig, shoot)
	}

	if util.NotNilOrEmpty(upgradeConfig.Purpose) {
		purpose := gardener_types.ShootPurpose(*upgradeConfig.Purpose)
		shoot.Spec.Purpose = &purpose
	}

	shoot.Spec.Maintenance.AutoUpdate.KubernetesVersion = upgradeConfig.EnableKubernetesVersionAutoUpdate
	shoot.Spec.Maintenance.AutoUpdate.MachineImageVersion = &upgradeConfig.EnableMachineImageVersionAutoUpdate

	if len(shoot.Spec.Provider.Workers) == 0 {
		return apperrors.Internal("no worker groups assigned to Gardener shoot '%s'", shoot.Name)
	}

	if util.NotNilOrEmpty(upgradeConfig.DiskType) {
		shoot.Spec.Provider.Workers[0].Volume.Type = upgradeConfig.DiskType
	}

	if upgradeConfig.VolumeSizeGB != nil {
		shoot.Spec.Provider.Workers[0].Volume.VolumeSize = fmt.Sprintf("%dGi", *upgradeConfig.VolumeSizeGB)
	}

	// We support only single working group during provisioning
	shoot.Spec.Provider.Workers[0].MaxSurge = util.PtrTo(intstr.FromInt(upgradeConfig.MaxSurge))
	shoot.Spec.Provider.Workers[0].MaxUnavailable = util.PtrTo(intstr.FromInt(upgradeConfig.MaxUnavailable))
	shoot.Spec.Provider.Workers[0].Machine.Type = upgradeConfig.MachineType
	shoot.Spec.Provider.Workers[0].Maximum = int32(upgradeConfig.AutoScalerMax)
	shoot.Spec.Provider.Workers[0].Minimum = int32(upgradeConfig.AutoScalerMin)
	if util.NotNilOrEmpty(upgradeConfig.MachineImage) {
		shoot.Spec.Provider.Workers[0].Machine.Image.Name = *upgradeConfig.MachineImage
	}
	if util.NotNilOrEmpty(upgradeConfig.MachineImageVersion) {
		shoot.Spec.Provider.Workers[0].Machine.Image.Version = upgradeConfig.MachineImageVersion
	}
	if upgradeConfig.OIDCConfig != nil {
		if shoot.Spec.Kubernetes.KubeAPIServer == nil {
			shoot.Spec.Kubernetes.KubeAPIServer = &gardener_types.KubeAPIServerConfig{}
		}
		shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig = &gardener_types.OIDCConfig{
			ClientID:       &upgradeConfig.OIDCConfig.ClientID,
			GroupsClaim:    &upgradeConfig.OIDCConfig.GroupsClaim,
			IssuerURL:      &upgradeConfig.OIDCConfig.IssuerURL,
			SigningAlgs:    upgradeConfig.OIDCConfig.SigningAlgs,
			UsernameClaim:  &upgradeConfig.OIDCConfig.UsernameClaim,
			UsernamePrefix: &upgradeConfig.OIDCConfig.UsernamePrefix,
		}
	}
	if util.NotNilOrEmpty(upgradeConfig.ExposureClassName) {
		shoot.Spec.ExposureClassName = upgradeConfig.ExposureClassName
	}

	if upgradeConfig.ShootNetworkingFilterDisabled != nil {
		upgradedExtensions := []gardener_types.Extension{}
		for _, extension := range shoot.Spec.Extensions {
			if extension.Type != ShootNetworkingFilterExtensionType {
				upgradedExtensions = append(upgradedExtensions, extension)
			}
		}
		upgradedExtensions = append(upgradedExtensions, gardener_types.Extension{
			Type:     ShootNetworkingFilterExtensionType,
			Disabled: upgradeConfig.ShootNetworkingFilterDisabled,
		})
		shoot.Spec.Extensions = upgradedExtensions
	}

	// Needed for upgrade to Kubernetes 1.25
	shoot.Spec.Kubernetes.AllowPrivilegedContainers = nil

	disablePlugin := true
	podSecurityPolicyPlugin := gardener_types.AdmissionPlugin{
		Name:     "PodSecurityPolicy",
		Disabled: &disablePlugin,
	}

	shoot.Spec.Kubernetes.KubeAPIServer.AdmissionPlugins = append(shoot.Spec.Kubernetes.KubeAPIServer.AdmissionPlugins, podSecurityPolicyPlugin)

	return nil
}

func adjustStaticKubeconfigFlag(upgradeConfig GardenerConfig, shoot *gardener_types.Shoot) {
	if upgradeConfig.KubernetesVersion != "" {
		var upgradedKubernetesVersion, _ = version.NewVersion(upgradeConfig.KubernetesVersion)
		var firstVersionNotSupportingStaticConfigs, _ = version.NewVersion("1.27.0")
		if upgradedKubernetesVersion.GreaterThanOrEqual(firstVersionNotSupportingStaticConfigs) {
			shoot.Spec.Kubernetes.EnableStaticTokenKubeconfig = util.PtrTo(false)
		}
	}
}

func getMachineConfig(config GardenerConfig) gardener_types.Machine {
	machine := gardener_types.Machine{
		Type: config.MachineType,
	}
	if util.NotNilOrEmpty(config.MachineImage) {
		machine.Image = &gardener_types.ShootMachineImage{
			Name: *config.MachineImage,
		}
		if util.NotNilOrEmpty(config.MachineImageVersion) {
			machine.Image.Version = config.MachineImageVersion
		}
	}
	return machine

}

func getAWSZonesNames(zones []*gqlschema.AWSZoneInput) []string {
	zoneNames := make([]string, 0)

	for _, zone := range zones {
		zoneNames = append(zoneNames, zone.Name)
	}
	return zoneNames
}

func getAzureZonesNames(zones []*gqlschema.AzureZoneInput) []string {
	zoneNames := make([]string, 0)

	for _, zone := range zones {
		zoneNames = append(zoneNames, fmt.Sprint(zone.Name))
	}
	return zoneNames
}
