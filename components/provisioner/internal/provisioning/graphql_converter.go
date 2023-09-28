package provisioning

import (
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

type GraphQLConverter interface {
	RuntimeStatusToGraphQLStatus(status model.RuntimeStatus) *gqlschema.RuntimeStatus
	OperationStatusToGQLOperationStatus(operation model.Operation) *gqlschema.OperationStatus
}

func NewGraphQLConverter() GraphQLConverter {
	return &graphQLConverter{}
}

type graphQLConverter struct{}

func (c graphQLConverter) RuntimeStatusToGraphQLStatus(status model.RuntimeStatus) *gqlschema.RuntimeStatus {
	return &gqlschema.RuntimeStatus{
		LastOperationStatus:     c.OperationStatusToGQLOperationStatus(status.LastOperationStatus),
		RuntimeConnectionStatus: c.runtimeConnectionStatusToGraphQLStatus(status.RuntimeConnectionStatus),
		RuntimeConfiguration:    c.clusterToToGraphQLRuntimeConfiguration(status.RuntimeConfiguration),
	}
}

func (c graphQLConverter) OperationStatusToGQLOperationStatus(operation model.Operation) *gqlschema.OperationStatus {
	return &gqlschema.OperationStatus{
		ID:        &operation.ID,
		Operation: c.operationTypeToGraphQLType(operation.Type),
		State:     c.operationStateToGraphQLState(operation.State),
		Message:   &operation.Message,
		RuntimeID: &operation.ClusterID,
		LastError: &gqlschema.LastError{
			ErrMessage: operation.ErrMessage,
			Reason:     operation.Reason,
			Component:  operation.Component,
		},
	}
}

func (c graphQLConverter) runtimeConnectionStatusToGraphQLStatus(status model.RuntimeAgentConnectionStatus) *gqlschema.RuntimeConnectionStatus {
	return &gqlschema.RuntimeConnectionStatus{Status: c.runtimeAgentConnectionStatusToGraphQLStatus(status)}
}

func (c graphQLConverter) runtimeAgentConnectionStatusToGraphQLStatus(status model.RuntimeAgentConnectionStatus) gqlschema.RuntimeAgentConnectionStatus {
	switch status {
	case model.RuntimeAgentConnectionStatusConnected:
		return gqlschema.RuntimeAgentConnectionStatusConnected
	case model.RuntimeAgentConnectionStatusDisconnected:
		return gqlschema.RuntimeAgentConnectionStatusDisconnected
	case model.RuntimeAgentConnectionStatusPending:
		return gqlschema.RuntimeAgentConnectionStatusPending
	default:
		return ""
	}
}

func (c graphQLConverter) clusterToToGraphQLRuntimeConfiguration(config model.Cluster) *gqlschema.RuntimeConfig {
	runtimeConfig := &gqlschema.RuntimeConfig{
		ClusterConfig: c.gardenerConfigToGraphQLConfig(config.ClusterConfig),
		Kubeconfig:    config.Kubeconfig,
	}
	if config.KymaConfig != nil {
		runtimeConfig.KymaConfig = c.kymaConfigToGraphQLConfig(*config.KymaConfig)
	}
	return runtimeConfig
}

func (c graphQLConverter) gardenerConfigToGraphQLConfig(config model.GardenerConfig) *gqlschema.GardenerConfig {

	var providerSpecificConfig gqlschema.ProviderSpecificConfig
	if config.GardenerProviderConfig != nil {
		providerSpecificConfig = config.GardenerProviderConfig.AsProviderSpecificConfig()
	}

	return &gqlschema.GardenerConfig{
		Name:                                &config.Name,
		KubernetesVersion:                   &config.KubernetesVersion,
		DiskType:                            config.DiskType,
		VolumeSizeGb:                        config.VolumeSizeGB,
		MachineType:                         &config.MachineType,
		MachineImage:                        config.MachineImage,
		MachineImageVersion:                 config.MachineImageVersion,
		Provider:                            &config.Provider,
		Purpose:                             config.Purpose,
		LicenceType:                         config.LicenceType,
		Seed:                                &config.Seed,
		TargetSecret:                        &config.TargetSecret,
		WorkerCidr:                          &config.WorkerCidr,
		PodsCidr:                            config.PodsCIDR,
		ServicesCidr:                        config.ServicesCIDR,
		Region:                              &config.Region,
		AutoScalerMin:                       &config.AutoScalerMin,
		AutoScalerMax:                       &config.AutoScalerMax,
		MaxSurge:                            &config.MaxSurge,
		MaxUnavailable:                      &config.MaxUnavailable,
		EnableKubernetesVersionAutoUpdate:   &config.EnableKubernetesVersionAutoUpdate,
		EnableMachineImageVersionAutoUpdate: &config.EnableMachineImageVersionAutoUpdate,
		ProviderSpecificConfig:              providerSpecificConfig,
		OidcConfig:                          c.oidcConfigToGraphQLConfig(config.OIDCConfig),
		DNSConfig:                           c.dnsConfigToGraphQLConfig(config.DNSConfig),
		ExposureClassName:                   config.ExposureClassName,
		ShootNetworkingFilterDisabled:       config.ShootNetworkingFilterDisabled,
		ControlPlaneFailureTolerance:        config.ControlPlaneFailureTolerance,
		EuAccess:                            &config.EuAccess,
	}
}

func (c graphQLConverter) oidcConfigToGraphQLConfig(config *model.OIDCConfig) *gqlschema.OIDCConfig {
	if config == nil {
		return nil
	}
	return &gqlschema.OIDCConfig{
		ClientID:       config.ClientID,
		GroupsClaim:    config.GroupsClaim,
		IssuerURL:      config.IssuerURL,
		SigningAlgs:    config.SigningAlgs,
		UsernameClaim:  config.UsernameClaim,
		UsernamePrefix: config.UsernamePrefix,
	}
}

func (c graphQLConverter) kymaConfigToGraphQLConfig(config model.KymaConfig) *gqlschema.KymaConfig {
	var components []*gqlschema.ComponentConfiguration
	for _, cmp := range config.Components {

		component := gqlschema.ComponentConfiguration{
			Component:     string(cmp.Component),
			Namespace:     cmp.Namespace,
			Configuration: c.configurationToGraphQLConfig(cmp.Configuration),
			SourceURL:     cmp.SourceURL,
		}

		components = append(components, &component)
	}

	return &gqlschema.KymaConfig{
		Profile:       c.profileToGraphQLProfile(config.Profile),
		Components:    components,
		Configuration: c.configurationToGraphQLConfig(config.GlobalConfiguration),
	}
}

func (c graphQLConverter) configurationToGraphQLConfig(cfg model.Configuration) []*gqlschema.ConfigEntry {
	configuration := make([]*gqlschema.ConfigEntry, 0, len(cfg.ConfigEntries))

	for _, configEntry := range cfg.ConfigEntries {
		secret := configEntry.Secret

		configuration = append(configuration, &gqlschema.ConfigEntry{
			Key:    configEntry.Key,
			Value:  configEntry.Value,
			Secret: &secret,
		})
	}

	return configuration
}

func (c graphQLConverter) operationTypeToGraphQLType(operationType model.OperationType) gqlschema.OperationType {
	switch operationType {
	case model.Provision:
		return gqlschema.OperationTypeProvision
	case model.ProvisionNoInstall:
		return gqlschema.OperationTypeProvision
	case model.Deprovision:
		return gqlschema.OperationTypeDeprovision
	case model.DeprovisionNoInstall:
		return gqlschema.OperationTypeDeprovisionNoInstall
	case model.Upgrade:
		return gqlschema.OperationTypeUpgrade
	case model.UpgradeShoot:
		return gqlschema.OperationTypeUpgradeShoot
	case model.ReconnectRuntime:
		return gqlschema.OperationTypeReconnectRuntime
	case model.Hibernate:
		return gqlschema.OperationTypeHibernate
	default:
		return ""
	}
}

func (c graphQLConverter) operationStateToGraphQLState(state model.OperationState) gqlschema.OperationState {
	switch state {
	case model.InProgress:
		return gqlschema.OperationStateInProgress
	case model.Succeeded:
		return gqlschema.OperationStateSucceeded
	case model.Failed:
		return gqlschema.OperationStateFailed
	default:
		return ""
	}
}

func (c graphQLConverter) profileToGraphQLProfile(profile *model.KymaProfile) *gqlschema.KymaProfile {

	if profile == nil {
		return nil
	}

	var result gqlschema.KymaProfile

	switch *profile {
	case model.EvaluationProfile:
		result = gqlschema.KymaProfileEvaluation
	case model.ProductionProfile:
		result = gqlschema.KymaProfileProduction
	default:
		result = gqlschema.KymaProfile("")
	}

	return &result
}

func (c graphQLConverter) dnsConfigToGraphQLConfig(config *model.DNSConfig) *gqlschema.DNSConfig {
	if config == nil {
		return nil
	}

	gqlConfig := gqlschema.DNSConfig{
		Domain: config.Domain,
	}

	for _, provider := range config.Providers {
		gqlConfig.Providers = append(gqlConfig.Providers,
			&gqlschema.DNSProvider{
				DomainsInclude: provider.DomainsInclude,
				Primary:        provider.Primary,
				SecretName:     provider.SecretName,
				Type:           provider.Type,
			},
		)
	}

	return &gqlConfig
}
