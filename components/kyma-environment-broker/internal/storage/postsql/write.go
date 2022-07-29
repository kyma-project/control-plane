package postsql

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"

	"github.com/gocraft/dbr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/lib/pq"
)

const (
	UniqueViolationErrorCode = "23505"
)

type writeSession struct {
	session     *dbr.Session
	transaction *dbr.Tx
}

func (ws writeSession) InsertInstance(instance dbmodel.InstanceDTO) dberr.Error {
	_, err := ws.insertInto(InstancesTableName).
		Pair("instance_id", instance.InstanceID).
		Pair("runtime_id", instance.RuntimeID).
		Pair("global_account_id", instance.GlobalAccountID).
		Pair("subscription_global_account_id", instance.SubscriptionGlobalAccountID).
		Pair("sub_account_id", instance.SubAccountID).
		Pair("service_id", instance.ServiceID).
		Pair("service_name", instance.ServiceName).
		Pair("service_plan_id", instance.ServicePlanID).
		Pair("service_plan_name", instance.ServicePlanName).
		Pair("dashboard_url", instance.DashboardURL).
		Pair("provisioning_parameters", instance.ProvisioningParameters).
		Pair("provider_region", instance.ProviderRegion).
		Pair("provider", instance.Provider).
		// in postgres database it will be equal to "0001-01-01 00:00:00+00"
		Pair("deleted_at", time.Time{}).
		Pair("expired_at", instance.ExpiredAt).
		Pair("version", instance.Version).
		Exec()

	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			if err.Code == UniqueViolationErrorCode {
				return dberr.AlreadyExists("operation with id %s already exist", instance.InstanceID)
			}
		}
		return dberr.Internal("Failed to insert record to Instance table: %s", err)
	}

	return nil
}

func (ws writeSession) DeleteInstance(instanceID string) dberr.Error {
	_, err := ws.deleteFrom(InstancesTableName).
		Where(dbr.Eq("instance_id", instanceID)).
		Exec()

	if err != nil {
		return dberr.Internal("Failed to delete record from Instance table: %s", err)
	}
	return nil
}

func (ws writeSession) UpdateInstance(instance dbmodel.InstanceDTO) dberr.Error {
	res, err := ws.update(InstancesTableName).
		Where(dbr.Eq("instance_id", instance.InstanceID)).
		Where(dbr.Eq("version", instance.Version)).
		Set("instance_id", instance.InstanceID).
		Set("runtime_id", instance.RuntimeID).
		Set("global_account_id", instance.GlobalAccountID).
		Set("subscription_global_account_id", instance.SubscriptionGlobalAccountID).
		Set("service_id", instance.ServiceID).
		Set("service_plan_id", instance.ServicePlanID).
		Set("dashboard_url", instance.DashboardURL).
		Set("provisioning_parameters", instance.ProvisioningParameters).
		Set("provider_region", instance.ProviderRegion).
		Set("provider", instance.Provider).
		Set("updated_at", time.Now()).
		Set("version", instance.Version+1).
		Set("expired_at", instance.ExpiredAt).
		Exec()
	if err != nil {
		return dberr.Internal("Failed to update record to Instance table: %s", err)
	}
	rAffected, err := res.RowsAffected()
	if err != nil {
		// the optimistic locking requires numbers of rows affected
		return dberr.Internal("the DB driver does not support RowsAffected operation")
	}
	if rAffected == int64(0) {
		return dberr.NotFound("Cannot find Instance with ID:'%s' Version: %v", instance.InstanceID, instance.Version)
	}

	return nil
}

func (ws writeSession) InsertOperation(op dbmodel.OperationDTO) dberr.Error {
	_, err := ws.insertInto(OperationTableName).
		Pair("id", op.ID).
		Pair("instance_id", op.InstanceID).
		Pair("version", op.Version).
		Pair("created_at", op.CreatedAt).
		Pair("updated_at", op.UpdatedAt).
		Pair("description", op.Description).
		Pair("state", op.State).
		Pair("target_operation_id", op.TargetOperationID).
		Pair("type", op.Type).
		Pair("data", op.Data).
		Pair("orchestration_id", op.OrchestrationID.String).
		Pair("provisioning_parameters", op.ProvisioningParameters.String).
		Pair("finished_stages", op.FinishedStages).
		Exec()

	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			if err.Code == UniqueViolationErrorCode {
				return dberr.AlreadyExists("operation with id %s already exist", op.ID)
			}
		}
		return dberr.Internal("Failed to insert record to operations table: %s", err)
	}

	return nil
}

func (ws writeSession) InsertOrchestration(o dbmodel.OrchestrationDTO) dberr.Error {
	_, err := ws.insertInto(OrchestrationTableName).
		Pair("orchestration_id", o.OrchestrationID).
		Pair("created_at", o.CreatedAt).
		Pair("updated_at", o.UpdatedAt).
		Pair("description", o.Description).
		Pair("state", o.State).
		Pair("type", o.Type).
		Pair("parameters", o.Parameters).
		Exec()

	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			if err.Code == UniqueViolationErrorCode {
				return dberr.AlreadyExists("Orchestration with id %s already exist", o.OrchestrationID)
			}
		}
		return dberr.Internal("Failed to insert record to orchestration table: %s", err)
	}

	return nil
}

func (ws writeSession) UpdateOrchestration(o dbmodel.OrchestrationDTO) dberr.Error {
	res, err := ws.update(OrchestrationTableName).
		Where(dbr.Eq("orchestration_id", o.OrchestrationID)).
		Set("created_at", o.CreatedAt).
		Set("updated_at", o.UpdatedAt).
		Set("description", o.Description).
		Set("state", o.State).
		Set("type", o.Type).
		Set("parameters", o.Parameters).
		Exec()

	if err != nil {
		if err == dbr.ErrNotFound {
			return dberr.NotFound("Cannot find Orchestration with ID:'%s'", o.OrchestrationID)
		}
		return dberr.Internal("Failed to update record to Orchestration table: %s", err)
	}
	rAffected, e := res.RowsAffected()
	if e != nil {
		// the optimistic locking requires numbers of rows affected
		return dberr.Internal("the DB driver does not support RowsAffected operation")
	}
	if rAffected == int64(0) {
		return dberr.NotFound("Cannot find Orchestration with ID:'%s'", o.OrchestrationID)
	}

	return nil
}

func (ws writeSession) InsertRuntimeState(state dbmodel.RuntimeStateDTO) dberr.Error {
	_, err := ws.insertInto(RuntimeStateTableName).
		Pair("id", state.ID).
		Pair("operation_id", state.OperationID).
		Pair("runtime_id", state.RuntimeID).
		Pair("created_at", state.CreatedAt).
		Pair("kyma_version", state.KymaVersion).
		Pair("k8s_version", state.K8SVersion).
		Pair("kyma_config", state.KymaConfig).
		Pair("cluster_config", state.ClusterConfig).
		Pair("cluster_setup", state.ClusterSetup).
		Exec()

	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			if err.Code == UniqueViolationErrorCode {
				return dberr.AlreadyExists("RuntimeState with id %s already exist", state.ID)
			}
		}
		return dberr.Internal("Failed to insert record to RuntimeState table: %s", err)
	}

	return nil
}

func (ws writeSession) UpdateOperation(op dbmodel.OperationDTO) dberr.Error {
	res, err := ws.update(OperationTableName).
		Where(dbr.Eq("id", op.ID)).
		Where(dbr.Eq("version", op.Version)).
		Set("instance_id", op.InstanceID).
		Set("version", op.Version+1).
		Set("created_at", op.CreatedAt).
		Set("updated_at", op.UpdatedAt).
		Set("description", op.Description).
		Set("state", op.State).
		Set("target_operation_id", op.TargetOperationID).
		Set("type", op.Type).
		Set("data", op.Data).
		Set("orchestration_id", op.OrchestrationID.String).
		Set("provisioning_parameters", op.ProvisioningParameters.String).
		Set("finished_stages", op.FinishedStages).
		Exec()

	if err != nil {
		if err == dbr.ErrNotFound {
			return dberr.NotFound("Cannot find Operation with ID:'%s'", op.ID)
		}
		return dberr.Internal("Failed to update record to Operation table: %s", err)
	}
	rAffected, e := res.RowsAffected()
	if e != nil {
		// the optimistic locking requires numbers of rows affected
		return dberr.Internal("the DB driver does not support RowsAffected operation")
	}
	if rAffected == int64(0) {
		return dberr.NotFound("Cannot find Operation with ID:'%s' Version: %v", op.ID, op.Version)
	}

	return nil
}

func (ws writeSession) Commit() dberr.Error {
	err := ws.transaction.Commit()
	if err != nil {
		return dberr.Internal("Failed to commit transaction: %s", err)
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
