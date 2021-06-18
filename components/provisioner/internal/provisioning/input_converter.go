package provisioning

import (
	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"

	"github.com/kyma-project/control-plane/components/provisioner/internal/installation/release"
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
	releaseProvider release.Provider,
	gardenerProject string,
	defaultEnableKubernetesVersionAutoUpdate,
	defaultEnableMachineImageVersionAutoUpdate,
	forceAllowPrivilegedContainers bool) InputConverter {

	return &converter{
		uuidGenerator:                              uuidGenerator,
		releaseProvider:                            releaseProvider,
		gardenerProject:                            gardenerProject,
		defaultEnableKubernetesVersionAutoUpdate:   defaultEnableKubernetesVersionAutoUpdate,
		defaultEnableMachineImageVersionAutoUpdate: defaultEnableMachineImageVersionAutoUpdate,
		forceAllowPrivilegedContainers:             forceAllowPrivilegedContainers,
	}
}

type converter struct {
	uuidGenerator                              uuid.UUIDGenerator
	releaseProvider                            release.Provider
	gardenerProject                            string
	defaultEnableKubernetesVersionAutoUpdate   bool
	defaultEnableMachineImageVersionAutoUpdate bool
	forceAllowPrivilegedContainers             bool
}

func (c converter) ProvisioningInputToCluster(runtimeID string, input gqlschema.ProvisionRuntimeInput, tenant, subAccountId string) (model.Cluster, apperrors.AppError) {
	var err apperrors.AppError

	var kymaConfig model.KymaConfig
	if input.KymaConfig != nil {
		kymaConfig, err = c.KymaConfigFromInput(runtimeID, *input.KymaConfig)
		if err != nil {
			return model.Cluster{}, err
		}
	}

	if input.ClusterConfig == nil || input.ClusterConfig.GardenerConfig == nil {
		return model.Cluster{}, apperrors.BadRequest("error: ClusterConfig not provided or GardenerConfig not provided")
	}

	gardenerConfigAllowPrivilegedContainers := c.shouldAllowPrivilegedContainers(
		input.ClusterConfig.GardenerConfig.AllowPrivilegedContainers,
		kymaConfig.Release.TillerYAML)

	gardenerConfig, err := c.gardenerConfigFromInput(
		runtimeID,
		input.ClusterConfig.GardenerConfig,
		gardenerConfigAllowPrivilegedContainers)
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

func (c converter) gardenerConfigFromInput(runtimeID string, input *gqlschema.GardenerConfigInput, allowPrivilegedContainers bool) (model.GardenerConfig, apperrors.AppError) {
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
		Seed:                                util.UnwrapStr(input.Seed),
		TargetSecret:                        input.TargetSecret,
		MachineType:                         input.MachineType,
		MachineImage:                        input.MachineImage,
		MachineImageVersion:                 input.MachineImageVersion,
		DiskType:                            input.DiskType,
		VolumeSizeGB:                        input.VolumeSizeGb,
		WorkerCidr:                          input.WorkerCidr,
		AutoScalerMin:                       input.AutoScalerMin,
		AutoScalerMax:                       input.AutoScalerMax,
		MaxSurge:                            input.MaxSurge,
		MaxUnavailable:                      input.MaxUnavailable,
		Purpose:                             input.Purpose,
		LicenceType:                         input.LicenceType,
		EnableKubernetesVersionAutoUpdate:   util.UnwrapBoolOrDefault(input.EnableKubernetesVersionAutoUpdate, c.defaultEnableKubernetesVersionAutoUpdate),
		EnableMachineImageVersionAutoUpdate: util.UnwrapBoolOrDefault(input.EnableMachineImageVersionAutoUpdate, c.defaultEnableMachineImageVersionAutoUpdate),
		AllowPrivilegedContainers:           allowPrivilegedContainers,
		ClusterID:                           runtimeID,
		GardenerProviderConfig:              providerSpecificConfig,
		OIDCConfig:                          oidcConfigFromInput(input.OidcConfig),
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

func (c converter) shouldAllowPrivilegedContainers(inputAllowPrivilegedContainers *bool, tillerYaml string) bool {
	if c.forceAllowPrivilegedContainers {
		return true
	}
	isTillerPresent := tillerYaml != ""
	return util.UnwrapBoolOrDefault(inputAllowPrivilegedContainers, isTillerPresent)
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
		ID:                        config.ID,
		ClusterID:                 config.ClusterID,
		Name:                      config.Name,
		ProjectName:               config.ProjectName,
		Provider:                  config.Provider,
		Seed:                      config.Seed,
		TargetSecret:              config.TargetSecret,
		Region:                    config.Region,
		LicenceType:               config.LicenceType,
		AllowPrivilegedContainers: config.AllowPrivilegedContainers,

		Purpose:                             util.DefaultStrIfNil(input.Purpose, config.Purpose),
		KubernetesVersion:                   util.UnwrapStrOrDefault(input.KubernetesVersion, config.KubernetesVersion),
		MachineType:                         util.UnwrapStrOrDefault(input.MachineType, config.MachineType),
		DiskType:                            util.DefaultStrIfNil(input.DiskType, config.DiskType),
		VolumeSizeGB:                        util.DefaultIntIfNil(input.VolumeSizeGb, config.VolumeSizeGB),
		MachineImage:                        util.DefaultStrIfNil(input.MachineImage, config.MachineImage),
		MachineImageVersion:                 util.DefaultStrIfNil(input.MachineImageVersion, config.MachineImageVersion),
		AutoScalerMin:                       util.UnwrapIntOrDefault(input.AutoScalerMin, config.AutoScalerMin),
		AutoScalerMax:                       util.UnwrapIntOrDefault(input.AutoScalerMax, config.AutoScalerMax),
		MaxSurge:                            util.UnwrapIntOrDefault(input.MaxSurge, config.MaxSurge),
		MaxUnavailable:                      util.UnwrapIntOrDefault(input.MaxUnavailable, config.MaxUnavailable),
		EnableKubernetesVersionAutoUpdate:   util.UnwrapBoolOrDefault(input.EnableKubernetesVersionAutoUpdate, config.EnableKubernetesVersionAutoUpdate),
		EnableMachineImageVersionAutoUpdate: util.UnwrapBoolOrDefault(input.EnableMachineImageVersionAutoUpdate, config.EnableMachineImageVersionAutoUpdate),
		GardenerProviderConfig:              providerSpecificConfig,
		OIDCConfig:                          oidcConfigFromInput(input.OidcConfig),
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
	kymaRelease, err := c.releaseProvider.GetReleaseByVersion(input.Version)
	if err != nil {
		return model.KymaConfig{}, apperrors.Internal("failed to get Kyma Release with version %s: %s", input.Version, err.Error())
	}

	var components []model.KymaComponentConfig
	kymaConfigID := c.uuidGenerator.New()

	for i, component := range input.Components {
		id := c.uuidGenerator.New()

		kymaConfigModule := model.KymaComponentConfig{
			ID:             id,
			Component:      model.KymaComponent(component.Component),
			Namespace:      component.Namespace,
			SourceURL:      component.SourceURL,
			Configuration:  c.configurationFromInput(component.Configuration),
			ComponentOrder: i + 1,
			KymaConfigID:   kymaConfigID,
		}

		components = append(components, kymaConfigModule)
	}

	return model.KymaConfig{
		ID:                  kymaConfigID,
		Release:             kymaRelease,
		Profile:             c.graphQLProfileToProfile(input.Profile),
		Components:          components,
		ClusterID:           runtimeID,
		GlobalConfiguration: c.configurationFromInput(input.Configuration),
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

func (c converter) configurationFromInput(input []*gqlschema.ConfigEntryInput) model.Configuration {
	configuration := model.Configuration{
		ConfigEntries: make([]model.ConfigEntry, 0, len(input)),
	}

	for _, ce := range input {
		configuration.ConfigEntries = append(configuration.ConfigEntries, configEntryFromInput(ce))
	}

	return configuration
}

func configEntryFromInput(entry *gqlschema.ConfigEntryInput) model.ConfigEntry {
	return model.NewConfigEntry(entry.Key, entry.Value, util.UnwrapBoolOrDefault(entry.Secret, false))
}
