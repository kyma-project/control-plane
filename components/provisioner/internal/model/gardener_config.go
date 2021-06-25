package model

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryRuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	SubAccountLabel = "subaccount"
	AccountLabel    = "account"

	LicenceTypeAnnotation = "kcp.provisioner.kyma-project.io/licence-type"
)

type OIDCConfig struct {
	ClientID       string   `json:"clientID"`
	GroupsClaim    string   `json:"groupsClaim"`
	IssuerURL      string   `json:"issuerURL"`
	SigningAlgs    []string `json:"signingAlgs"`
	UsernameClaim  string   `json:"usernameClaim"`
	UsernamePrefix string   `json:"usernamePrefix"`
}

type GardenerConfig struct {
	ID                                  string
	ClusterID                           string
	Name                                string
	ProjectName                         string
	KubernetesVersion                   string
	VolumeSizeGB                        *int
	DiskType                            *string
	MachineType                         string
	MachineImage                        *string
	MachineImageVersion                 *string
	Provider                            string
	Purpose                             *string
	LicenceType                         *string
	Seed                                string
	TargetSecret                        string
	Region                              string
	WorkerCidr                          string
	AutoScalerMin                       int
	AutoScalerMax                       int
	MaxSurge                            int
	MaxUnavailable                      int
	EnableKubernetesVersionAutoUpdate   bool
	EnableMachineImageVersionAutoUpdate bool
	AllowPrivilegedContainers           bool
	GardenerProviderConfig              GardenerProviderConfig
	OIDCConfig                          *OIDCConfig
}

func (c GardenerConfig) ToShootTemplate(namespace string, accountId string, subAccountId string, oidcConfig *OIDCConfig) (*gardener_types.Shoot, apperrors.AppError) {
	enableBasicAuthentication := false

	var seed *string = nil
	if c.Seed != "" {
		seed = util.StringPtr(c.Seed)
	}
	var purpose *gardener_types.ShootPurpose = nil
	if util.NotNilOrEmpty(c.Purpose) {
		p := gardener_types.ShootPurpose(*c.Purpose)
		purpose = &p
	}

	annotations := make(map[string]string)
	if c.LicenceType != nil {
		annotations[LicenceTypeAnnotation] = *c.LicenceType
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
			SecretBindingName: c.TargetSecret,
			SeedName:          seed,
			Region:            c.Region,
			Kubernetes: gardener_types.Kubernetes{
				AllowPrivilegedContainers: &c.AllowPrivilegedContainers,
				Version:                   c.KubernetesVersion,
				KubeAPIServer: &gardener_types.KubeAPIServerConfig{
					EnableBasicAuthentication: &enableBasicAuthentication,
					OIDCConfig:                gardenerOidcConfig(oidcConfig),
				},
			},
			Networking: gardener_types.Networking{
				Type:  "calico",                        // Default value - we may consider adding it to API (if Hydroform will support it)
				Nodes: util.StringPtr("10.250.0.0/19"), // TODO: it is required - provide configuration in API (when Hydroform will support it)
			},
			Purpose: purpose,
			Maintenance: &gardener_types.Maintenance{
				AutoUpdate: &gardener_types.MaintenanceAutoUpdate{
					KubernetesVersion:   c.EnableKubernetesVersionAutoUpdate,
					MachineImageVersion: c.EnableMachineImageVersionAutoUpdate,
				},
			},
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

type ProviderSpecificConfig string

func (c ProviderSpecificConfig) RawJSON() string {
	return string(c)
}

type GardenerProviderConfig interface {
	RawJSON() string
	AsProviderSpecificConfig() gqlschema.ProviderSpecificConfig
	ExtendShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError
	EditShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError
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

	// needed for backward compatibility - originally, AWS clusters were created only with single AZ based on SingleZoneAWSProviderConfigInput schema
	// TODO: Remove after data migration
	var singleZoneAwsProviderConfig SingleZoneAWSProviderConfigInput
	err = util.DecodeJson(jsonData, &singleZoneAwsProviderConfig)
	if err == nil {
		awsProviderConfig := gqlschema.AWSProviderConfigInput{
			VpcCidr: singleZoneAwsProviderConfig.VpcCidr,
			AwsZones: []*gqlschema.AWSZoneInput{
				{
					Name:         singleZoneAwsProviderConfig.Zone,
					PublicCidr:   singleZoneAwsProviderConfig.PublicCidr,
					InternalCidr: singleZoneAwsProviderConfig.InternalCidr,
					WorkerCidr:   "10.250.0.0/19",
				},
			},
		}

		var jsonData bytes.Buffer
		err = util.Encode(awsProviderConfig, &jsonData)
		if err == nil {
			return &AWSGardenerConfig{input: &awsProviderConfig, ProviderSpecificConfig: ProviderSpecificConfig(jsonData.String())}, nil
		}
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

func (c GCPGardenerConfig) AsProviderSpecificConfig() gqlschema.ProviderSpecificConfig {
	return gqlschema.GCPProviderConfig{Zones: c.input.Zones}
}

func (c GCPGardenerConfig) EditShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError {
	return updateShootConfig(gardenerConfig, shoot, c.input.Zones)
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

func (c AzureGardenerConfig) AsProviderSpecificConfig() gqlschema.ProviderSpecificConfig {
	return gqlschema.AzureProviderConfig{VnetCidr: &c.input.VnetCidr, Zones: c.input.Zones}
}

type AWSGardenerConfig struct {
	ProviderSpecificConfig
	input *gqlschema.AWSProviderConfigInput `db:"-"`
}

func (c AzureGardenerConfig) EditShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError {
	return updateShootConfig(gardenerConfig, shoot, c.input.Zones)
}

func (c AzureGardenerConfig) ExtendShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError {
	shoot.Spec.CloudProfileName = "az"

	workers := []gardener_types.Worker{getWorkerConfig(gardenerConfig, c.input.Zones)}

	azInfra := NewAzureInfrastructure(gardenerConfig.WorkerCidr, c)
	jsonData, err := json.Marshal(azInfra)
	if err != nil {
		return apperrors.Internal("error encoding infrastructure config: %s", err.Error())
	}

	azureControlPlane := NewAzureControlPlane(c.input.Zones)
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

func (c AWSGardenerConfig) EditShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError {
	zoneNames := getAWSZonesNames(c.input.AwsZones)
	return updateShootConfig(gardenerConfig, shoot, zoneNames)
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

func (c OpenStackGardenerConfig) AsProviderSpecificConfig() gqlschema.ProviderSpecificConfig {
	return gqlschema.OpenStackProviderConfig{
		Zones:                c.input.Zones,
		FloatingPoolName:     c.input.FloatingPoolName,
		CloudProfileName:     c.input.CloudProfileName,
		LoadBalancerProvider: c.input.LoadBalancerProvider,
	}
}

func (c OpenStackGardenerConfig) EditShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError {
	return updateShootConfig(gardenerConfig, shoot, c.input.Zones)
}

func (c OpenStackGardenerConfig) ExtendShootConfig(gardenerConfig GardenerConfig, shoot *gardener_types.Shoot) apperrors.AppError {
	shoot.Spec.CloudProfileName = c.input.CloudProfileName

	workers := []gardener_types.Worker{getWorkerConfig(gardenerConfig, c.input.Zones)}

	openStackInfra := NewOpenStackInfrastructure(c.input.FloatingPoolName, gardenerConfig.WorkerCidr)
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
		MaxSurge:       util.IntOrStringPtr(intstr.FromInt(gardenerConfig.MaxSurge)),
		MaxUnavailable: util.IntOrStringPtr(intstr.FromInt(gardenerConfig.MaxUnavailable)),
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

func updateShootConfig(upgradeConfig GardenerConfig, shoot *gardener_types.Shoot, zones []string) apperrors.AppError {

	if upgradeConfig.KubernetesVersion != "" {
		shoot.Spec.Kubernetes.Version = upgradeConfig.KubernetesVersion
	}

	if util.NotNilOrEmpty(upgradeConfig.Purpose) {
		purpose := gardener_types.ShootPurpose(*upgradeConfig.Purpose)
		shoot.Spec.Purpose = &purpose
	}

	shoot.Spec.Maintenance.AutoUpdate.KubernetesVersion = upgradeConfig.EnableKubernetesVersionAutoUpdate
	shoot.Spec.Maintenance.AutoUpdate.MachineImageVersion = upgradeConfig.EnableMachineImageVersionAutoUpdate

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
	shoot.Spec.Provider.Workers[0].MaxSurge = util.IntOrStringPtr(intstr.FromInt(upgradeConfig.MaxSurge))
	shoot.Spec.Provider.Workers[0].MaxUnavailable = util.IntOrStringPtr(intstr.FromInt(upgradeConfig.MaxUnavailable))
	shoot.Spec.Provider.Workers[0].Machine.Type = upgradeConfig.MachineType
	shoot.Spec.Provider.Workers[0].Maximum = int32(upgradeConfig.AutoScalerMax)
	shoot.Spec.Provider.Workers[0].Minimum = int32(upgradeConfig.AutoScalerMin)
	shoot.Spec.Provider.Workers[0].Zones = zones
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
	return nil
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

// SingleZoneAWSProviderConfigInput describes old schema with only single AZ available for AWS clusters
// TODO: remove after data migration
type SingleZoneAWSProviderConfigInput struct {
	Zone         string `json:"zone"`
	VpcCidr      string `json:"vpcCidr"`
	PublicCidr   string `json:"publicCidr"`
	InternalCidr string `json:"internalCidr"`
}
