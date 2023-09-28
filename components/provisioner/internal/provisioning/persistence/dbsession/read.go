package dbsession

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/gocraft/dbr/v2"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
)

type readSession struct {
	session *dbr.Session
	decrypt decryptFunc
}

func (r readSession) GetTenant(runtimeID string) (string, dberrors.Error) {
	var tenant string

	err := r.session.
		Select("tenant").
		From("cluster").
		Where(dbr.Eq("cluster.id", runtimeID)).
		LoadOne(&tenant)

	if err != nil {
		if err == dbr.ErrNotFound {
			return "", dberrors.NotFound("Cannot find Tenant for runtimeID:'%s", runtimeID)
		}

		return "", dberrors.Internal("Failed to get Tenant: %s", err)
	}
	return tenant, nil
}

func (r readSession) GetTenantForOperation(operationID string) (string, dberrors.Error) {
	var tenant string

	err := r.session.
		Select("cluster.tenant").
		From("operation").
		Join("cluster", "operation.cluster_id=cluster.id").
		Where(dbr.Eq("operation.id", operationID)).
		LoadOne(&tenant)

	if err != nil {
		if err == dbr.ErrNotFound {
			return "", dberrors.NotFound("Cannot find Tenant for operationID:'%s", operationID)
		}

		return "", dberrors.Internal("Failed to get Tenant: %s", err)
	}
	return tenant, nil
}

func (r readSession) GetCluster(runtimeID string) (model.Cluster, dberrors.Error) {
	var cluster model.Cluster

	err := r.session.
		Select(
			"id", "kubeconfig", "tenant",
			"creation_timestamp", "deleted", "sub_account_id",
			"active_kyma_config_id", "is_kubeconfig_encrypted").
		From("cluster").
		Where(dbr.Eq("cluster.id", runtimeID)).
		LoadOne(&cluster)

	if err != nil {
		if err == dbr.ErrNotFound {
			return model.Cluster{}, dberrors.NotFound("Cannot find Cluster for runtimeID: %s", runtimeID)
		}
		return model.Cluster{}, dberrors.Internal("Failed to get Cluster: %s", err)
	}

	if cluster.IsKubeconfigEncrypted {
		decryptedClusterKubeconfig, dberr := r.decryptKubeconfig(cluster.Kubeconfig)
		if dberr != nil {
			return model.Cluster{}, dberr.Append("Cannot decrypt Kubeconfig for runtimeID: %s", runtimeID)
		}
		cluster.Kubeconfig = decryptedClusterKubeconfig
	}

	providerConfig, dberr := r.getGardenerConfig(runtimeID)
	if dberr != nil {
		return model.Cluster{}, dberr.Append("Cannot get Provider config for runtimeID: %s", runtimeID)
	}
	cluster.ClusterConfig = providerConfig

	oidcConfig, dberr := r.getOidcConfig(providerConfig.ID)
	if dberr != nil {
		return model.Cluster{}, dberr.Append("Cannot get Oidc config for runtimeID: %s", runtimeID)
	}
	cluster.ClusterConfig.OIDCConfig = &oidcConfig

	dnsConfig, dberr := r.getDNSConfig(providerConfig.ID)
	if dberr != nil {
		return model.Cluster{}, dberr.Append("Cannot get DNS config for runtimeID: %s", runtimeID)
	}
	cluster.ClusterConfig.DNSConfig = dnsConfig

	if cluster.ActiveKymaConfigId != nil {
		kymaConfig, dberr := r.getKymaConfig(runtimeID, *cluster.ActiveKymaConfigId)
		if dberr != nil {
			return model.Cluster{}, dberr.Append("Cannot get Kyma config for runtimeID: %s", runtimeID)
		}
		cluster.KymaConfig = &kymaConfig
	}

	clusterAdministrators, dberr := r.getClusterAdministrators(runtimeID)
	if dberr != nil {
		return model.Cluster{}, dberr.Append("Cannot get Cluster administrators for runtimeID: %s", runtimeID)
	}
	cluster.Administrators = make([]string, len(clusterAdministrators))
	for i := range clusterAdministrators {
		cluster.Administrators[i] = clusterAdministrators[i].UserId
	}

	return cluster, nil
}

func (r readSession) GetGardenerClusterByName(name string) (model.Cluster, dberrors.Error) {
	var clusterWithProvider = struct {
		model.Cluster
		gardenerConfigRead
	}{}

	err := r.session.
		Select(
			"cluster.id", "cluster.kubeconfig", "cluster.tenant",
			"cluster.creation_timestamp", "cluster.deleted", "cluster.active_kyma_config_id",
			"name", "project_name", "kubernetes_version",
			"volume_size_gb", "disk_type", "machine_type", "machine_image", "machine_image_version",
			"provider", "purpose", "seed", "target_secret", "worker_cidr", "pods_cidr", "services_cidr", "region", "auto_scaler_min",
			"auto_scaler_max", "max_surge", "max_unavailable", "enable_kubernetes_version_auto_update",
			"enable_machine_image_version_auto_update", "provider_specific_config",
			"shoot_networking_filter_disabled", "control_plane_failure_tolerance").
		From("gardener_config").
		Join("cluster", "gardener_config.cluster_id=cluster.id").
		Where(dbr.Eq("name", name)).
		LoadOne(&clusterWithProvider)

	if err != nil {
		if err == dbr.ErrNotFound {
			return model.Cluster{}, dberrors.NotFound("Cannot find Gardener Cluster with name: %s", name)
		}

		return model.Cluster{}, dberrors.Internal("Failed to get Gardener Cluster with name: %s, error: %s", name, err)
	}
	cluster := clusterWithProvider.Cluster

	err = clusterWithProvider.gardenerConfigRead.DecodeProviderConfig()
	if err != nil {
		return model.Cluster{}, dberrors.Internal("Failed to decode Gardener provider config fetched from database: %s", err.Error())
	}
	cluster.ClusterConfig = clusterWithProvider.gardenerConfigRead.GardenerConfig

	if cluster.ActiveKymaConfigId != nil {
		kymaConfig, dberr := r.getKymaConfig(clusterWithProvider.Cluster.ID, *cluster.ActiveKymaConfigId)
		if dberr != nil {
			return model.Cluster{}, dberr.Append("Cannot get Kyma config for runtimeID: %s", clusterWithProvider.Cluster.ID)
		}
		cluster.KymaConfig = &kymaConfig
	}

	return cluster, nil
}

type kymaComponentConfigDTO struct {
	ID                  string
	KymaConfigID        string
	GlobalConfiguration []byte
	ReleaseID           string
	Profile             *string
	Version             string
	Component           string
	Namespace           string
	SourceURL           *string
	Configuration       []byte
	ComponentOrder      *int
	ClusterID           string
}

type kymaConfigDTO []kymaComponentConfigDTO

func (c kymaConfigDTO) parseToKymaConfig(runtimeID string) (model.KymaConfig, dberrors.Error) {
	kymaModulesOrdered := make(map[int][]model.KymaComponentConfig, 0)

	for _, componentCfg := range c {
		var configuration model.Configuration
		err := json.Unmarshal(componentCfg.Configuration, &configuration)
		if err != nil {
			return model.KymaConfig{}, dberrors.Internal("Failed to unmarshal configuration for %s component: %s", componentCfg.Component, err.Error())
		}

		kymaComponentConfig := model.KymaComponentConfig{
			ID:             componentCfg.ID,
			Component:      model.KymaComponent(componentCfg.Component),
			Namespace:      componentCfg.Namespace,
			SourceURL:      componentCfg.SourceURL,
			Configuration:  configuration,
			KymaConfigID:   componentCfg.KymaConfigID,
			ComponentOrder: util.UnwrapInt(componentCfg.ComponentOrder),
		}

		// In case order is 0 for all components map stores slice (it is the case for Runtimes created before migration)
		kymaModulesOrdered[util.UnwrapInt(componentCfg.ComponentOrder)] = append(kymaModulesOrdered[util.UnwrapInt(componentCfg.ComponentOrder)], kymaComponentConfig)
	}

	keys := make([]int, 0, len(kymaModulesOrdered))
	for k, _ := range kymaModulesOrdered {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	orderedComponents := make([]model.KymaComponentConfig, 0, len(c))
	for _, k := range keys {
		orderedComponents = append(orderedComponents, kymaModulesOrdered[k]...)
	}

	var globalConfiguration model.Configuration
	err := json.Unmarshal(c[0].GlobalConfiguration, &globalConfiguration)
	if err != nil {
		return model.KymaConfig{}, dberrors.Internal("Failed to unmarshal global configuration: %s", err.Error())
	}

	var kymaProfile *model.KymaProfile
	if c[0].Profile != nil {
		profile := model.KymaProfile(*c[0].Profile)
		kymaProfile = &profile
	}

	return model.KymaConfig{
		ID:                  c[0].KymaConfigID,
		Profile:             kymaProfile,
		Components:          orderedComponents,
		GlobalConfiguration: globalConfiguration,
		ClusterID:           runtimeID,
	}, nil
}

func (r readSession) getKymaConfig(runtimeID, kymaConfigId string) (model.KymaConfig, dberrors.Error) {
	var kymaConfig kymaConfigDTO

	rowsCount, err := r.session.
		Select("kyma_config_id", "kyma_config.release_id", "kyma_config.profile", "kyma_config.global_configuration",
			"kyma_component_config.id", "kyma_component_config.component", "kyma_component_config.namespace",
			"kyma_component_config.source_url", "kyma_component_config.configuration",
			"kyma_component_config.component_order",
			"cluster_id",
			"kyma_release.version", "kyma_release.tiller_yaml", "kyma_release.installer_yaml").
		From("cluster").
		Join("kyma_config", "cluster.id=kyma_config.cluster_id").
		Join("kyma_component_config", "kyma_config.id=kyma_component_config.kyma_config_id").
		Join("kyma_release", "kyma_config.release_id=kyma_release.id").
		Where(dbr.Eq("kyma_config.id", kymaConfigId)).
		Load(&kymaConfig)

	if err != nil {
		return model.KymaConfig{}, dberrors.Internal("Failed to get Kyma Config: %s", err)
	}

	if rowsCount == 0 {
		return model.KymaConfig{}, dberrors.NotFound("Cannot find Kyma Config for runtimeID: %s", runtimeID)
	}

	return kymaConfig.parseToKymaConfig(runtimeID)
}

func (r readSession) getClusterAdministrators(runtimeID string) ([]model.ClusterAdministrator, dberrors.Error) {
	var clusterAdministrators []model.ClusterAdministrator

	_, err := r.session.
		Select("*").
		From("cluster_Administrator").
		Where(dbr.Eq("cluster_id", runtimeID)).
		Load(&clusterAdministrators)

	if err != nil {
		return []model.ClusterAdministrator{}, dberrors.Internal("Failed to get Cluster Administrators: %s", err)
	}

	decryptedClusterAdministrators, err := r.decryptClusterAdministrators(clusterAdministrators)
	if err != nil {
		return []model.ClusterAdministrator{}, dberrors.Internal("Failed to decrypt Cluster Administrators: %s", err)
	}

	return decryptedClusterAdministrators, nil
}

type gardenerConfigRead struct {
	model.GardenerConfig
	ProviderSpecificConfig string `db:"provider_specific_config"`
}

func (gcr *gardenerConfigRead) DecodeProviderConfig() error {
	gardenerConfigProviderConfig, err := model.NewGardenerProviderConfigFromJSON(gcr.ProviderSpecificConfig)
	if err != nil {
		return fmt.Errorf("error decoding Gardener provider config: %s", err.Error())
	}

	gcr.GardenerProviderConfig = gardenerConfigProviderConfig
	return nil
}

func (r readSession) getGardenerConfig(runtimeID string) (model.GardenerConfig, dberrors.Error) {
	gardenerConfig := gardenerConfigRead{}

	err := r.session.
		Select("gardener_config.id", "cluster_id", "gardener_config.name", "project_name",
			"kubernetes_version", "volume_size_gb", "disk_type", "machine_type", "machine_image",
			"machine_image_version", "provider", "purpose", "seed", "target_secret", "worker_cidr", "pods_cidr", "services_cidr", "region",
			"auto_scaler_min", "auto_scaler_max", "max_surge", "max_unavailable",
			"enable_kubernetes_version_auto_update", "enable_machine_image_version_auto_update",
			"exposure_class_name", "provider_specific_config",
			"shoot_networking_filter_disabled", "control_plane_failure_tolerance", "eu_access").
		From("cluster").
		Join("gardener_config", "cluster.id=gardener_config.cluster_id").
		Where(dbr.Eq("cluster.id", runtimeID)).
		LoadOne(&gardenerConfig)

	if err != nil {
		if err == dbr.ErrNotFound {
			return model.GardenerConfig{}, dberrors.NotFound("Gardener config for %s Runtime not found: %s", runtimeID, err.Error())
		}

		return model.GardenerConfig{}, dberrors.Internal("Failed to get Gardener config for %s Runtime: %s", runtimeID, err.Error())
	}

	err = gardenerConfig.DecodeProviderConfig()
	if err != nil {
		return model.GardenerConfig{}, dberrors.Internal("Failed to decode Gardener provider config fetched from database: %s", err.Error())
	}

	return gardenerConfig.GardenerConfig, nil
}

var (
	operationColumns = []string{
		"id", "type", "start_timestamp", "stage", "end_timestamp", "state", "message", "cluster_id", "last_transition", "err_message", "reason", "component",
	}
)

func (r readSession) GetOperation(operationID string) (model.Operation, dberrors.Error) {
	var operation model.Operation

	err := r.session.
		Select(operationColumns...).
		From("operation").
		Where(dbr.Eq("id", operationID)).
		LoadOne(&operation)

	if err != nil {
		if err == dbr.ErrNotFound {
			return model.Operation{}, dberrors.NotFound("Operation not found for id: %s", operationID)
		}
		return model.Operation{}, dberrors.Internal("Failed to get %s operation: %s", operationID, err)
	}

	return operation, nil
}

func (r readSession) GetLastOperation(runtimeID string) (model.Operation, dberrors.Error) {
	lastOperationDateSelect := r.session.
		Select("MAX(start_timestamp)").
		From("operation").
		Where(dbr.Eq("cluster_id", runtimeID))

	var operation model.Operation

	err := r.session.
		Select(operationColumns...).
		From("operation").
		Where(dbr.Eq("start_timestamp", lastOperationDateSelect)).
		LoadOne(&operation)

	if err != nil {
		if err == dbr.ErrNotFound {
			return model.Operation{}, dberrors.NotFound("Last operation not found for runtime: %s", runtimeID)
		}
		return model.Operation{}, dberrors.Internal("Failed to get last operation: %s", err)
	}

	return operation, nil
}

func (r readSession) ListInProgressOperations() ([]model.Operation, dberrors.Error) {
	var operations []model.Operation

	_, err := r.session.
		Select(operationColumns...).
		From("operation").
		Where(dbr.Eq("state", model.InProgress)).
		Load(&operations)

	if err != nil {
		if err == dbr.ErrNotFound {
			return []model.Operation{}, nil
		}
		return nil, dberrors.Internal("Failed to list In Progress operation: %s", err)
	}

	return operations, nil
}

func (r readSession) GetRuntimeUpgrade(operationId string) (model.RuntimeUpgrade, dberrors.Error) {
	var runtimeUpgrade model.RuntimeUpgrade

	_, err := r.session.
		Select("id", "state", "operation_id", "pre_upgrade_kyma_config_id", "post_upgrade_kyma_config_id").
		From("runtime_upgrade").
		Where(dbr.Eq("operation_id", operationId)).
		Load(&runtimeUpgrade)

	if err != nil {
		if err == dbr.ErrNotFound {
			return model.RuntimeUpgrade{}, dberrors.NotFound("Runtime upgrade not found for operation with %s id", operationId)
		}
		return model.RuntimeUpgrade{}, dberrors.Internal("Failed to get Runtime upgrade for operation %s: %s", operationId, err)
	}

	return runtimeUpgrade, nil
}

func (r readSession) InProgressOperationsCount() (model.OperationsCount, dberrors.Error) {
	var opsCount []struct {
		Type  model.OperationType
		Count int
	}

	_, err := r.session.Select("type", "count(*)").
		From("operation").
		Where(dbr.Eq("state", model.InProgress)).
		GroupBy("type").
		Load(&opsCount)

	if err != nil {
		if err == dbr.ErrNotFound {
			return model.OperationsCount{}, dberrors.NotFound("Operations not found: %s", err.Error())
		}
		return model.OperationsCount{}, dberrors.Internal("Failed to count operations in progress: %s", err.Error())
	}

	operationsCount := model.OperationsCount{
		Count: make(map[model.OperationType]int, len(opsCount)),
	}
	for _, op := range opsCount {
		operationsCount.Count[op.Type] = op.Count
	}

	return operationsCount, nil
}

func (r readSession) getOidcConfig(gardenerConfigID string) (model.OIDCConfig, dberrors.Error) {
	var oidc model.OIDCConfig
	var algorithms []string

	_, err := r.session.
		Select("*").
		From("oidc_config").
		Where(dbr.Eq("gardener_config_id", gardenerConfigID)).
		Load(&oidc)

	if err != nil {
		return model.OIDCConfig{}, dberrors.Internal("Failed to get oidc: %s", err)
	}

	_, err = r.session.
		Select("algorithm").
		From("signing_algorithms").
		Where(dbr.Eq("oidc_config_id", gardenerConfigID)).
		Load(&algorithms)

	if err != nil {
		return model.OIDCConfig{}, dberrors.Internal("Failed to get algorithm: %s", err)
	}

	oidc.SigningAlgs = algorithms

	return oidc, nil
}

func (r readSession) getDNSConfig(gardenerConfigID string) (*model.DNSConfig, dberrors.Error) {
	var dnsConfigWithID struct {
		model.DNSConfig
		ID string `db:"id"`
	}
	var dnsProvidersPreSplit []struct {
		model.DNSProvider
		RawDomains string `db:"domains_include"`
	}

	err := r.session.
		Select("domain", "id").
		From("dns_config").
		Where(dbr.Eq("gardener_config_id", gardenerConfigID)).
		LoadOne(&dnsConfigWithID)

	if errors.Is(err, dbr.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, dberrors.Internal("Failed to get DNS config: %s", err)
	}

	dnsConfig := dnsConfigWithID.DNSConfig

	_, err = r.session.
		Select("is_primary", "secret_name", "type", "domains_include").
		From("dns_providers").
		Where(dbr.Eq("dns_config_id", dnsConfigWithID.ID)).
		Load(&dnsProvidersPreSplit)

	if err != nil {
		return nil, dberrors.Internal("Failed to get DNS provider: %s", err)
	}

	for _, provider := range dnsProvidersPreSplit {
		provider.DNSProvider.DomainsInclude = strings.Split(provider.RawDomains, ",")
		dnsConfig.Providers = append(dnsConfig.Providers, &provider.DNSProvider)
	}

	return &dnsConfig, nil
}

func (r readSession) decryptKubeconfig(encryptedKubeconfig *string) (*string, dberrors.Error) {
	if encryptedKubeconfig == nil {
		return nil, nil
	}
	decryptedKubeconfigSlice, err := r.decrypt([]byte(*encryptedKubeconfig))
	if err != nil {
		return nil, dberrors.Internal("failed to decrypt kubeconfig: %v", err)
	}
	decryptedKubeconfig := string(decryptedKubeconfigSlice)
	return &decryptedKubeconfig, nil
}

func (r readSession) decryptClusterAdministrators(
	encryptedClusterAdministrators []model.ClusterAdministrator) (
	[]model.ClusterAdministrator, dberrors.Error) {

	var decryptedClusterAdministrators []model.ClusterAdministrator
	for _, ea := range encryptedClusterAdministrators {
		if ea.IsUserIdEncrypted {
			decryptedUserID, err := r.decrypt([]byte(ea.UserId))
			if err != nil {
				return nil, dberrors.Internal("failed to decrypt user ID: %v", err)
			}
			decryptedClusterAdministrator := model.ClusterAdministrator{
				ID:        ea.ID,
				ClusterId: ea.ClusterId,
				UserId:    string(decryptedUserID),
			}
			decryptedClusterAdministrators = append(decryptedClusterAdministrators, decryptedClusterAdministrator)
		} else {
			decryptedClusterAdministrators = append(decryptedClusterAdministrators, ea)
		}
	}
	return decryptedClusterAdministrators, nil
}
