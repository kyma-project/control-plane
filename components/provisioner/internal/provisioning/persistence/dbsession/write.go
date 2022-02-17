package dbsession

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	dbr "github.com/gocraft/dbr/v2"
	uuid "github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"
)

type writeSession struct {
	session     *dbr.Session
	transaction *dbr.Tx
}

//TODO: Remove after schema migration
func (ws writeSession) UpdateProviderSpecificConfig(id string, providerSpecificConfig string) dberrors.Error {
	res, err := ws.update("gardener_config").
		Where(dbr.Eq("id", id)).
		Set("provider_specific_config", providerSpecificConfig).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to update provider_specific_config for gardener shoot cluster '%s': %s", id, err)
	}

	return ws.updateSucceeded(res, fmt.Sprintf("Failed to update provider_specific_config for gardener shoot cluster '%s' state: %s", id, err))
}

//TODO: Remove after schema migration
func (ws writeSession) InsertRelease(artifacts model.Release) dberrors.Error {
	_, err := ws.insertInto("kyma_release").
		Columns("id", "version", "tiller_yaml", "installer_yaml").
		Record(artifacts).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to insert record to Release table: %s", err)
	}

	return nil
}

func (ws writeSession) InsertCluster(cluster model.Cluster) dberrors.Error {
	var kymaConfigId *string
	if cluster.KymaConfig != nil {
		kymaConfigId = &cluster.KymaConfig.ID
	}

	_, err := ws.insertInto("cluster").
		Pair("id", cluster.ID).
		Pair("creation_timestamp", cluster.CreationTimestamp).
		Pair("tenant", cluster.Tenant).
		Pair("sub_account_id", cluster.SubAccountId).
		Pair("active_kyma_config_id", kymaConfigId). // Possible due to deferred constrain
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to insert record to Cluster table: %s", err)
	}

	dbErr := ws.InsertAdministrators(cluster.ID, cluster.Administrators)
	if dbErr != nil {
		return dbErr
	}

	return nil
}

func (ws writeSession) InsertAdministrators(clusterId string, administrators []string) dberrors.Error {
	_, err := ws.deleteFrom("cluster_administrator").
		Where(dbr.Eq("cluster_id", clusterId)).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to delete record to cluster_administrator table: %s", err)
	}

	for _, admin := range administrators {
		_, err := ws.insertInto("cluster_administrator").
			Pair("id", uuid.New().String()).
			Pair("cluster_id", clusterId).
			Pair("user_id", admin).Exec()

		if err != nil {
			return dberrors.Internal("Failed to insert record to cluster_administrator table: %s", err)
		}
	}

	return nil
}

func (ws writeSession) InsertGardenerConfig(config model.GardenerConfig) dberrors.Error {
	_, err := ws.insertInto("gardener_config").
		Pair("id", config.ID).
		Pair("cluster_id", config.ClusterID).
		Pair("project_name", config.ProjectName).
		Pair("name", config.Name).
		Pair("kubernetes_version", config.KubernetesVersion).
		Pair("volume_size_gb", config.VolumeSizeGB).
		Pair("machine_type", config.MachineType).
		Pair("machine_image", config.MachineImage).
		Pair("machine_image_version", config.MachineImageVersion).
		Pair("region", config.Region).
		Pair("provider", config.Provider).
		Pair("purpose", config.Purpose).
		Pair("licence_type", config.LicenceType).
		Pair("seed", config.Seed).
		Pair("target_secret", config.TargetSecret).
		Pair("disk_type", config.DiskType).
		Pair("worker_cidr", config.WorkerCidr).
		Pair("auto_scaler_min", config.AutoScalerMin).
		Pair("auto_scaler_max", config.AutoScalerMax).
		Pair("max_surge", config.MaxSurge).
		Pair("max_unavailable", config.MaxUnavailable).
		Pair("enable_kubernetes_version_auto_update", config.EnableKubernetesVersionAutoUpdate).
		Pair("enable_machine_image_version_auto_update", config.EnableMachineImageVersionAutoUpdate).
		Pair("allow_privileged_containers", config.AllowPrivilegedContainers).
		Pair("exposure_class_name", config.ExposureClassName).
		Pair("provider_specific_config", config.GardenerProviderConfig.RawJSON()).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to insert record to GardenerConfig table: %s", err)
	}

	if config.OIDCConfig != nil {
		err := ws.insertOidcConfig(config)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ws writeSession) insertOidcConfig(config model.GardenerConfig) dberrors.Error {
	_, err := ws.insertInto("oidc_config").
		Pair("id", config.ID).
		Pair("client_id", config.OIDCConfig.ClientID).
		Pair("groups_claim", config.OIDCConfig.GroupsClaim).
		Pair("issuer_url", config.OIDCConfig.IssuerURL).
		Pair("username_claim", config.OIDCConfig.UsernameClaim).
		Pair("username_prefix", config.OIDCConfig.UsernamePrefix).
		Pair("gardener_config_id", config.ID).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to insert record to OIDCConfig table: %s", err)
	}

	for _, algorithm := range config.OIDCConfig.SigningAlgs {
		_, err = ws.insertInto("signing_algorithms").
			Pair("id", uuid.New().String()).
			Pair("oidc_config_id", config.ID).
			Pair("algorithm", algorithm).
			Exec()

		if err != nil {
			return dberrors.Internal("Failed to insert record to SigningAlgorithms table: %s", err)
		}
	}
	return nil
}

func (ws writeSession) UpdateGardenerClusterConfig(config model.GardenerConfig) dberrors.Error {
	res, err := ws.update("gardener_config").
		Where(dbr.Eq("cluster_id", config.ClusterID)).
		Set("kubernetes_version", config.KubernetesVersion).
		Set("purpose", config.Purpose).
		Set("seed", config.Seed).
		Set("machine_type", config.MachineType).
		Set("machine_image", config.MachineImage).
		Set("machine_image_version", config.MachineImageVersion).
		Set("disk_type", config.DiskType).
		Set("volume_size_gb", config.VolumeSizeGB).
		Set("auto_scaler_min", config.AutoScalerMin).
		Set("auto_scaler_max", config.AutoScalerMax).
		Set("max_surge", config.MaxSurge).
		Set("max_unavailable", config.MaxUnavailable).
		Set("enable_kubernetes_version_auto_update", config.EnableKubernetesVersionAutoUpdate).
		Set("enable_machine_image_version_auto_update", config.EnableMachineImageVersionAutoUpdate).
		Set("exposure_class_name", config.ExposureClassName).
		Set("provider_specific_config", config.GardenerProviderConfig.RawJSON()).
		Exec()

	if config.OIDCConfig != nil {
		err = ws.updateOidcConfig(config)
		if err != nil {
			return dberrors.Internal("Failed to update record for oidc config %s", err)
		}
	}

	if err != nil {
		return dberrors.Internal("Failed to update record of configuration for gardener shoot cluster '%s': %s", config.Name, err)
	}

	return ws.updateSucceeded(res, fmt.Sprintf("Failed to update record of configuration for gardener shoot cluster '%s' state: %s", config.Name, err))
}

func (ws writeSession) updateOidcConfig(config model.GardenerConfig) dberrors.Error {
	_, err := ws.deleteFrom("oidc_config").
		Where(dbr.Eq("gardener_config_id", config.ID)).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to delete record to OIDCConfig table: %s", err)
	}

	_, err = ws.insertInto("oidc_config").
		Pair("id", config.ID).
		Pair("client_id", config.OIDCConfig.ClientID).
		Pair("groups_claim", config.OIDCConfig.GroupsClaim).
		Pair("issuer_url", config.OIDCConfig.IssuerURL).
		Pair("username_claim", config.OIDCConfig.UsernameClaim).
		Pair("username_prefix", config.OIDCConfig.UsernamePrefix).
		Pair("gardener_config_id", config.ID).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to update record to OIDCConfig table: %s", err)
	}

	_, err = ws.deleteFrom("signing_algorithms").
		Where(dbr.Eq("oidc_config_id", config.ID)).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to delete records from SigningAlgorithms table: %s", err)
	}

	for _, algorithm := range config.OIDCConfig.SigningAlgs {

		_, err = ws.insertInto("signing_algorithms").
			Pair("id", uuid.New().String()).
			Pair("oidc_config_id", config.ID).
			Pair("algorithm", algorithm).
			Exec()

		if err != nil {
			return dberrors.Internal("Failed to insert record to SigningAlgorithms table: %s", err)
		}
	}
	return nil
}

func (ws writeSession) InsertKymaConfig(kymaConfig model.KymaConfig) dberrors.Error {
	jsonConfig, err := json.Marshal(kymaConfig.GlobalConfiguration)
	if err != nil {
		return dberrors.Internal("Failed to marshal global configuration: %s", err.Error())
	}

	_, err = ws.insertInto("kyma_config").
		Pair("id", kymaConfig.ID).
		Pair("release_id", kymaConfig.Release.Id).
		Pair("profile", kymaConfig.Profile).
		Pair("cluster_id", kymaConfig.ClusterID).
		Pair("global_configuration", jsonConfig).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to insert record to KymaConfig table: %s", err)
	}

	for _, kymaConfigModule := range kymaConfig.Components {
		err = ws.insertKymaComponentConfig(kymaConfigModule)
		if err != nil {
			return dberrors.Internal("Failed to insert record to KymaComponentConfig table: %s", err)
		}
	}

	return nil
}

func (ws writeSession) insertKymaComponentConfig(kymaConfigModule model.KymaComponentConfig) dberrors.Error {
	jsonConfig, err := json.Marshal(kymaConfigModule.Configuration)
	if err != nil {
		return dberrors.Internal("Failed to marshal %s component configuration: %s", kymaConfigModule.Component, err.Error())
	}

	_, err = ws.insertInto("kyma_component_config").
		Pair("id", kymaConfigModule.ID).
		Pair("component", kymaConfigModule.Component).
		Pair("namespace", kymaConfigModule.Namespace).
		Pair("source_url", kymaConfigModule.SourceURL).
		Pair("kyma_config_id", kymaConfigModule.KymaConfigID).
		Pair("configuration", jsonConfig).
		Pair("component_order", &kymaConfigModule.ComponentOrder).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to insert record to KymaComponentConfig table: %s", err)
	}

	return nil
}

func (ws writeSession) InsertOperation(operation model.Operation) dberrors.Error {
	_, err := ws.insertInto("operation").
		Columns(operationColumns...).
		Record(operation).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to insert record to Type table: %s", err)
	}

	return nil
}

func (ws writeSession) DeleteCluster(runtimeID string) dberrors.Error {
	result, err := ws.deleteFrom("cluster").
		Where(dbr.Eq("id", runtimeID)).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to delete record in Cluster table: %s", err)
	}

	val, err := result.RowsAffected()

	if err != nil {
		return dberrors.Internal("Could not fetch the number of rows affected: %s", err)
	}

	if val == 0 {
		return dberrors.NotFound("Runtime with ID %s not found", runtimeID)
	}

	return nil
}

func (ws writeSession) UpdateOperationState(operationID string, message string, state model.OperationState, endTime time.Time) dberrors.Error {
	res, err := ws.update("operation").
		Where(dbr.Eq("id", operationID)).
		Set("state", state).
		Set("message", message).
		Set("end_timestamp", endTime).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to update operation %s state: %s", operationID, err)
	}

	return ws.updateSucceeded(res, fmt.Sprintf("Failed to update operation %s state: %s", operationID, err))
}

func (ws writeSession) UpdateOperationLastError(operationID, msg, reason, component string) dberrors.Error {
	res, err := ws.update("operation").
		Where(dbr.Eq("id", operationID)).
		Set("error_msg", msg).
		Set("error_reason", reason).
		Set("error_component", component).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to update operation %s last error: %s", operationID, err)
	}

	return ws.updateSucceeded(res, fmt.Sprintf("Failed to update operation %s last error: %s", operationID, err))
}

func (ws writeSession) TransitionOperation(operationID string, message string, stage model.OperationStage, transitionTime time.Time) dberrors.Error {
	res, err := ws.update("operation").
		Where(dbr.Eq("id", operationID)).
		Set("stage", stage).
		Set("message", message).
		Set("last_transition", transitionTime).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to update operation %s stage: %s", operationID, err)
	}

	return ws.updateSucceeded(res, fmt.Sprintf("Failed to update operation %s state: %s", operationID, err))
}

// Clean up this code when not needed (https://github.com/kyma-project/control-plane/issues/1371)
func (ws writeSession) FixShootProvisioningStage(message string, newStage model.OperationStage, transitionTime time.Time) dberrors.Error {
	legacyStageCondition := dbr.Eq("stage", "ShootProvisioning")
	provisioningOperation := dbr.Eq("type", model.Provision)
	inProgressOperation := dbr.Eq("state", model.InProgress)

	_, err := ws.update("operation").
		Where(dbr.And(legacyStageCondition, provisioningOperation, inProgressOperation)).
		Set("stage", newStage).
		Set("message", message).
		Set("last_transition", transitionTime).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to set stage: %v for operations", err)
	}

	return nil
}

func (ws writeSession) UpdateKubeconfig(runtimeID string, kubeconfig string) dberrors.Error {
	res, err := ws.update("cluster").
		Where(dbr.Eq("id", runtimeID)).
		Set("kubeconfig", kubeconfig).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to update cluster %s state: %s", runtimeID, err)
	}

	return ws.updateSucceeded(res, fmt.Sprintf("Failed to update cluster %s data: %s", runtimeID, err))
}

func (ws writeSession) SetActiveKymaConfig(runtimeID string, kymaConfigId string) dberrors.Error {
	res, err := ws.update("cluster").
		Where(dbr.Eq("id", runtimeID)).
		Set("active_kyma_config_id", kymaConfigId).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to update cluster %s Kyma config: %s", runtimeID, err)
	}

	return ws.updateSucceeded(res, fmt.Sprintf("Failed to update cluster %s kyma config: %s", runtimeID, err))
}

func (ws writeSession) UpdateUpgradeState(operationID string, upgradeState model.UpgradeState) dberrors.Error {
	res, err := ws.update("runtime_upgrade").
		Where(dbr.Eq("operation_id", operationID)).
		Set("state", upgradeState).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to update operation %s upgrade state: %s", operationID, err)
	}

	return ws.updateSucceeded(res, fmt.Sprintf("Failed to update operation %s upgrade state: %s", operationID, err))
}

func (ws writeSession) UpdateKubernetesVersion(runtimeID string, version string) dberrors.Error {
	res, err := ws.update("gardener_config").
		Where(dbr.Eq("cluster_id", runtimeID)).
		Set("kubernetes_version", version).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to update Kubernetes version in %s cluster: %s", runtimeID, err)
	}

	return ws.updateSucceeded(res, fmt.Sprintf("Failed to update Kubernetes version in %s cluster: %s", runtimeID, err))
}

func (ws writeSession) MarkClusterAsDeleted(runtimeID string) dberrors.Error {
	res, err := ws.update("cluster").
		Where(dbr.Eq("id", runtimeID)).
		Set("deleted", true).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to update cluster %s state: %s", runtimeID, err)
	}

	return ws.updateSucceeded(res, fmt.Sprintf("Failed to update cluster %s data: %s", runtimeID, err))
}

func (ws writeSession) InsertRuntimeUpgrade(runtimeUpgrade model.RuntimeUpgrade) dberrors.Error {
	_, err := ws.insertInto("runtime_upgrade").
		Columns("id", "state", "operation_id", "pre_upgrade_kyma_config_id", "post_upgrade_kyma_config_id").
		Record(runtimeUpgrade).
		Exec()
	if err != nil {
		return dberrors.Internal("Failed to insert Runtime Upgrade: %s", err.Error())
	}

	return nil
}

func (ws writeSession) UpdateTenant(runtimeID, tenant string) dberrors.Error {
	res, err := ws.update("cluster").
		Where(dbr.Eq("id", runtimeID)).
		Set("tenant", tenant).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to update cluster %s state: %s", runtimeID, err)
	}
	return ws.updateSucceeded(res, fmt.Sprintf("Failed to update tenant %s: %s", tenant, err))
}

func (ws writeSession) updateSucceeded(result sql.Result, errorMsg string) dberrors.Error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dberrors.Internal("Failed to get number of rows affected: %s", err)
	}

	if rowsAffected == 0 {
		return dberrors.NotFound(errorMsg)
	}

	return nil
}

func (ws writeSession) Commit() dberrors.Error {
	err := ws.transaction.Commit()
	if err != nil {
		return dberrors.Internal("Failed to commit transaction: %s", err)
	}

	return nil
}

func (ws writeSession) RollbackUnlessCommitted() {
	ws.transaction.RollbackUnlessCommitted()
}

func (ws writeSession) insertInto(table string) *dbr.InsertStmt {
	if ws.transaction != nil {
		return ws.transaction.InsertInto(table)
	}

	return ws.session.InsertInto(table)
}

func (ws writeSession) deleteFrom(table string) *dbr.DeleteStmt {
	if ws.transaction != nil {
		return ws.transaction.DeleteFrom(table)
	}

	return ws.session.DeleteFrom(table)
}

func (ws writeSession) update(table string) *dbr.UpdateStmt {
	if ws.transaction != nil {
		return ws.transaction.Update(table)
	}

	return ws.session.Update(table)
}
