package provisioning

import (
	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"

	"github.com/kyma-project/control-plane/components/provisioner/internal/util"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/uuid"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

type InputConverter interface {
	ProvisioningInputToCluster(runtimeID string, input gqlschema.ProvisionRuntimeInput, tenant, subAccountId string) (model.Cluster, apperrors.AppError)
	KymaConfigFromInput(runtimeID string, input gqlschema.KymaConfigInput) (model.KymaConfig, apperrors.AppError)
	UpgradeShootInputToGardenerConfig(input gqlschema.GardenerUpgradeInput, existing model.GardenerConfig) (model.GardenerConfig, apperrors.AppError)
}

func NewInputConverter(
	uuidGenerator uuid.UUIDGenerator,
	gardenerProject string,
	defaultEnableKubernetesVersionAutoUpdate,
	defaultEnableMachineImageVersionAutoUpdate bool) InputConverter {

	return &converter{
		uuidGenerator:                                    uuidGenerator,
		gardenerProject:                                  gardenerProject,
		defaultEnableKubernetesVersionAutoUpdate:         defaultEnableKubernetesVersionAutoUpdate,
		defaultEnableMachineImageVersionAutoUpdate:       defaultEnableMachineImageVersionAutoUpdate,
		defaultProvisioningShootNetworkingFilterDisabled: true,
		defaultEuAccess:                                  false,
	}
}

type converter struct {
	uuidGenerator                                    uuid.UUIDGenerator
	gardenerProject                                  string
	defaultEnableKubernetesVersionAutoUpdate         bool
	defaultEnableMachineImageVersionAutoUpdate       bool
	defaultProvisioningShootNetworkingFilterDisabled bool
	defaultEuAccess                                  bool
}

func (c converter) ProvisioningInputToCluster(runtimeID string, input gqlschema.ProvisionRuntimeInput, tenant, subAccountId string) (model.Cluster, apperrors.AppError) {
	var err apperrors.AppError

	var kymaConfig *model.KymaConfig

	if input.KymaConfig != nil {
		config, err := c.KymaConfigFromInput(runtimeID, *input.KymaConfig)
		kymaConfig = &config
		if err != nil {
			return model.Cluster{}, err
		}
	}

	if input.ClusterConfig == nil || input.ClusterConfig.GardenerConfig == nil {
		return model.Cluster{}, apperrors.BadRequest("error: ClusterConfig not provided or GardenerConfig not provided")
	}

	if input.ClusterConfig.GardenerConfig.ShootNetworkingFilterDisabled == nil {
		input.ClusterConfig.GardenerConfig.ShootNetworkingFilterDisabled = util.PtrTo(c.defaultProvisioningShootNetworkingFilterDisabled)
	}

	gardenerConfig, err := c.gardenerConfigFromInput(
		runtimeID,
		input.ClusterConfig.GardenerConfig)
	if err != nil {
		return model.Cluster{}, err
	}

	return model.Cluster{
		ID:             runtimeID,
		KymaConfig:     kymaConfig,
		ClusterConfig:  gardenerConfig,
		Tenant:         tenant,
		SubAccountId:   &subAccountId,
		Administrators: input.ClusterConfig.Administrators,
	}, nil
}

func (c converter) gardenerConfigFromInput(runtimeID string, input *gqlschema.GardenerConfigInput) (model.GardenerConfig, apperrors.AppError) {
	providerSpecificConfig, err := c.providerSpecificConfigFromInput(input.ProviderSpecificConfig)
	if err != nil {
		return model.GardenerConfig{}, err
	}

	id := c.uuidGenerator.New()
	return model.GardenerConfig{
		ID:                                  id,
		Name:                                input.Name,
		ProjectName:                         c.gardenerProject,
		KubernetesVersion:                   input.KubernetesVersion,
		Provider:                            input.Provider,
		Region:                              input.Region,
		Seed:                                util.UnwrapOrZero(input.Seed),
		TargetSecret:                        input.TargetSecret,
		MachineType:                         input.MachineType,
		MachineImage:                        input.MachineImage,
		MachineImageVersion:                 input.MachineImageVersion,
		DiskType:                            input.DiskType,
		VolumeSizeGB:                        input.VolumeSizeGb,
		WorkerCidr:                          input.WorkerCidr,
		PodsCIDR:                            input.PodsCidr,
		ServicesCIDR:                        input.ServicesCidr,
		AutoScalerMin:                       input.AutoScalerMin,
		AutoScalerMax:                       input.AutoScalerMax,
		MaxSurge:                            input.MaxSurge,
		MaxUnavailable:                      input.MaxUnavailable,
		Purpose:                             input.Purpose,
		LicenceType:                         input.LicenceType,
		EnableKubernetesVersionAutoUpdate:   util.UnwrapOrDefault(input.EnableKubernetesVersionAutoUpdate, c.defaultEnableKubernetesVersionAutoUpdate),
		EnableMachineImageVersionAutoUpdate: util.UnwrapOrDefault(input.EnableMachineImageVersionAutoUpdate, c.defaultEnableMachineImageVersionAutoUpdate),
		ClusterID:                           runtimeID,
		GardenerProviderConfig:              providerSpecificConfig,
		OIDCConfig:                          oidcConfigFromInput(input.OidcConfig),
		DNSConfig:                           dnsConfigFromInput(input.DNSConfig),
		ExposureClassName:                   input.ExposureClassName,
		ShootNetworkingFilterDisabled:       input.ShootNetworkingFilterDisabled,
		ControlPlaneFailureTolerance:        input.ControlPlaneFailureTolerance,
		EuAccess:                            util.UnwrapOrDefault(input.EuAccess, c.defaultEuAccess),
	}, nil
}

func oidcConfigFromInput(config *gqlschema.OIDCConfigInput) *model.OIDCConfig {
	if config != nil {
		return &model.OIDCConfig{
			ClientID:       config.ClientID,
			GroupsClaim:    config.GroupsClaim,
			IssuerURL:      config.IssuerURL,
			SigningAlgs:    config.SigningAlgs,
			UsernameClaim:  config.UsernameClaim,
			UsernamePrefix: config.UsernamePrefix,
		}
	}
	return nil
}

func dnsConfigFromInput(input *gqlschema.DNSConfigInput) *model.DNSConfig {
	config := model.DNSConfig{}
	if input != nil {
		config.Domain = input.Domain

		if len(input.Providers) != 0 {
			for _, v := range input.Providers {
				config.Providers = append(config.Providers, &model.DNSProvider{
					// after KEB fix it - restore original code
					// DomainsInclude: v.DomainsInclude,
					DomainsInclude: []string{input.Domain},
					Primary:        v.Primary,
					SecretName:     v.SecretName,
					Type:           v.Type,
				})
			}
		}

		return &config
	}

	return nil
}

func (c converter) UpgradeShootInputToGardenerConfig(input gqlschema.GardenerUpgradeInput, config model.GardenerConfig) (model.GardenerConfig, apperrors.AppError) {
	var providerSpecificConfig model.GardenerProviderConfig
	var err apperrors.AppError

	if input.ProviderSpecificConfig != nil {
		providerSpecificConfig, err = c.providerSpecificConfigFromInput(input.ProviderSpecificConfig)
		if providerSpecificConfig == nil {
			return model.GardenerConfig{}, err.Append("error converting provider specific config from input: %s", err)
		}
	} else {
		providerSpecificConfig = config.GardenerProviderConfig
	}

	return model.GardenerConfig{
		ID:           config.ID,
		ClusterID:    config.ClusterID,
		Name:         config.Name,
		ProjectName:  config.ProjectName,
		Provider:     config.Provider,
		Seed:         config.Seed,
		TargetSecret: config.TargetSecret,
		Region:       config.Region,
		LicenceType:  config.LicenceType,
		WorkerCidr:   config.WorkerCidr,

		Purpose:                             util.OkOrDefault(input.Purpose, config.Purpose),
		KubernetesVersion:                   util.UnwrapOrDefault(input.KubernetesVersion, config.KubernetesVersion),
		MachineType:                         util.UnwrapOrDefault(input.MachineType, config.MachineType),
		DiskType:                            util.OkOrDefault(input.DiskType, config.DiskType),
		VolumeSizeGB:                        util.OkOrDefault(input.VolumeSizeGb, config.VolumeSizeGB),
		MachineImage:                        util.OkOrDefault(input.MachineImage, config.MachineImage),
		MachineImageVersion:                 util.OkOrDefault(input.MachineImageVersion, config.MachineImageVersion),
		AutoScalerMin:                       util.UnwrapOrDefault(input.AutoScalerMin, config.AutoScalerMin),
		AutoScalerMax:                       util.UnwrapOrDefault(input.AutoScalerMax, config.AutoScalerMax),
		MaxSurge:                            util.UnwrapOrDefault(input.MaxSurge, config.MaxSurge),
		MaxUnavailable:                      util.UnwrapOrDefault(input.MaxUnavailable, config.MaxUnavailable),
		EnableKubernetesVersionAutoUpdate:   util.UnwrapOrDefault(input.EnableKubernetesVersionAutoUpdate, config.EnableKubernetesVersionAutoUpdate),
		EnableMachineImageVersionAutoUpdate: util.UnwrapOrDefault(input.EnableMachineImageVersionAutoUpdate, config.EnableMachineImageVersionAutoUpdate),
		GardenerProviderConfig:              providerSpecificConfig,
		OIDCConfig:                          oidcConfigFromInput(input.OidcConfig),
		ExposureClassName:                   util.OkOrDefault(input.ExposureClassName, config.ExposureClassName),
		ShootNetworkingFilterDisabled:       util.OkOrDefault(input.ShootNetworkingFilterDisabled, config.ShootNetworkingFilterDisabled),
	}, nil
}

func (c converter) providerSpecificConfigFromInput(input *gqlschema.ProviderSpecificInput) (model.GardenerProviderConfig, apperrors.AppError) {
	if input == nil {
		return nil, apperrors.Internal("provider config not specified")
	}

	if input.GcpConfig != nil {
		return model.NewGCPGardenerConfig(input.GcpConfig)
	}
	if input.AzureConfig != nil {
		return model.NewAzureGardenerConfig(input.AzureConfig)
	}
	if input.AwsConfig != nil {
		return model.NewAWSGardenerConfig(input.AwsConfig)
	}
	if input.OpenStackConfig != nil {
		return model.NewOpenStackGardenerConfig(input.OpenStackConfig)
	}

	return nil, apperrors.BadRequest("provider config not specified")
}

func (c converter) KymaConfigFromInput(runtimeID string, input gqlschema.KymaConfigInput) (model.KymaConfig, apperrors.AppError) {
	var components []model.KymaComponentConfig
	kymaConfigID := c.uuidGenerator.New()

	for i, component := range input.Components {
		id := c.uuidGenerator.New()

		kymaConfigModule := model.KymaComponentConfig{
			ID:             id,
			Component:      model.KymaComponent(component.Component),
			Namespace:      component.Namespace,
			SourceURL:      component.SourceURL,
			Configuration:  c.configurationFromInput(component.Configuration, component.ConflictStrategy),
			ComponentOrder: i + 1,
			KymaConfigID:   kymaConfigID,
		}

		components = append(components, kymaConfigModule)
	}

	return model.KymaConfig{
		ID:                  kymaConfigID,
		Profile:             c.graphQLProfileToProfile(input.Profile),
		Components:          components,
		ClusterID:           runtimeID,
		GlobalConfiguration: c.configurationFromInput(input.Configuration, input.ConflictStrategy),
	}, nil
}

func (c converter) graphQLProfileToProfile(profile *gqlschema.KymaProfile) *model.KymaProfile {
	if profile == nil {
		return nil
	}

	var result model.KymaProfile

	switch *profile {
	case gqlschema.KymaProfileEvaluation:
		result = model.EvaluationProfile
	case gqlschema.KymaProfileProduction:
		result = model.ProductionProfile
	default:
		result = model.KymaProfile("")
	}

	return &result

}

func (c converter) configurationFromInput(input []*gqlschema.ConfigEntryInput, conflict *gqlschema.ConflictStrategy) model.Configuration {
	configuration := model.Configuration{
		ConfigEntries: make([]model.ConfigEntry, 0, len(input)),
	}

	if conflict != nil {
		configuration.ConflictStrategy = conflict.String()
	}

	for _, ce := range input {
		configuration.ConfigEntries = append(configuration.ConfigEntries, configEntryFromInput(ce))
	}

	return configuration
}

func configEntryFromInput(entry *gqlschema.ConfigEntryInput) model.ConfigEntry {
	return model.NewConfigEntry(entry.Key, entry.Value, util.UnwrapOrDefault(entry.Secret, false))
}
