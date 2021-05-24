package dbconnection

import (
	"database/sql"
	"fmt"
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/model"
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/persistence/dberrors"

	"github.com/gocraft/dbr/v2"
)

type writeSession struct {
	session     *dbr.Session
	transaction *dbr.Tx
}

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
		Pair("provider_specific_config", config.GardenerProviderConfig).
		Exec()

	if err != nil {
		return dberrors.Internal("Failed to insert record to GardenerConfig table: %s", err)
	}

	return nil
}

func (ws writeSession) RollbackUnlessCommitted() {
	ws.transaction.RollbackUnlessCommitted()
}

func (ws writeSession) Commit() dberrors.Error {
	err := ws.transaction.Commit()
	if err != nil {
		return dberrors.Internal("Failed to commit transaction: %s", err)
	}

	return nil
}

func (ws writeSession) insertInto(table string) *dbr.InsertStmt {
	if ws.transaction != nil {
		return ws.transaction.InsertInto(table)
	}

	return ws.session.InsertInto(table)
}

func (ws writeSession) update(table string) *dbr.UpdateStmt {
	if ws.transaction != nil {
		return ws.transaction.Update(table)
	}

	return ws.session.Update(table)
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
