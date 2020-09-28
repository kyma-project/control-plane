package dbsession

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/pagination"

	"github.com/pkg/errors"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/predicate"

	"github.com/gocraft/dbr"
	"github.com/pivotal-cf/brokerapi/v7/domain"
)

type readSession struct {
	session *dbr.Session
}

func (r readSession) getInstancesJoinedWithOperationStatement() *dbr.SelectStmt {
	join := fmt.Sprintf("%s.instance_id = %s.instance_id", postsql.InstancesTableName, postsql.OperationTableName)
	stmt := r.session.
		Select("instances.instance_id, instances.runtime_id, instances.global_account_id, instances.service_id, instances.service_plan_id, instances.dashboard_url, instances.provisioning_parameters, instances.created_at, instances.updated_at, instances.deleted_at, instances.sub_account_id, instances.service_name, instances.service_plan_name, operations.state, operations.description, operations.type").
		From(postsql.InstancesTableName).
		LeftJoin(postsql.OperationTableName, join)
	return stmt
}

func (r readSession) FindAllInstancesJoinedWithOperation(prct ...predicate.Predicate) ([]internal.InstanceWithOperation, dberr.Error) {
	var instances []internal.InstanceWithOperation

	stmt := r.getInstancesJoinedWithOperationStatement()
	for _, p := range prct {
		p.ApplyToPostgres(stmt)
	}

	if _, err := stmt.Load(&instances); err != nil {
		return nil, dberr.Internal("Failed to fetch all instances: %s", err)
	}

	return instances, nil
}

func (r readSession) GetInstanceByID(instanceID string) (internal.Instance, dberr.Error) {
	var instance internal.Instance

	err := r.session.
		Select("*").
		From(postsql.InstancesTableName).
		Where(dbr.Eq("instance_id", instanceID)).
		LoadOne(&instance)

	if err != nil {
		if err == dbr.ErrNotFound {
			return internal.Instance{}, dberr.NotFound("Cannot find Instance for instanceID:'%s'", instanceID)
		}
		return internal.Instance{}, dberr.Internal("Failed to get Instance: %s", err)
	}
	return instance, nil
}

func (r readSession) FindAllInstancesForRuntimes(runtimeIdList []string) ([]internal.Instance, dberr.Error) {
	var instances []internal.Instance

	err := r.session.
		Select("*").
		From(postsql.InstancesTableName).
		Where("runtime_id IN ?", runtimeIdList).
		LoadOne(&instances)

	if err != nil {
		if err == dbr.ErrNotFound {
			return []internal.Instance{}, dberr.NotFound("Cannot find Instances for runtime ID list: '%v'", runtimeIdList)
		}
		return []internal.Instance{}, dberr.Internal("Failed to get Instances: %s", err)
	}
	return instances, nil
}

func (r readSession) FindAllInstancesForSubAccounts(subAccountslist []string) ([]internal.Instance, dberr.Error) {
	var instances []internal.Instance

	err := r.session.
		Select("*").
		From(postsql.InstancesTableName).
		Where("sub_account_id IN ?", subAccountslist).
		LoadOne(&instances)

	if err != nil {
		if err == dbr.ErrNotFound {
			return []internal.Instance{}, nil
		}
		return []internal.Instance{}, dberr.Internal("Failed to get Instances: %s", err)
	}
	return instances, nil
}

func (r readSession) GetOperationByID(opID string) (dbmodel.OperationDTO, dberr.Error) {
	condition := dbr.Eq("id", opID)
	operation, err := r.getOperation(condition)
	if err != nil {
		switch {
		case dberr.IsNotFound(err):
			return dbmodel.OperationDTO{}, dberr.NotFound("for ID: %s %s", opID, err)
		default:
			return dbmodel.OperationDTO{}, err
		}
	}
	return operation, nil
}

func (r readSession) GetOrchestrationByID(oID string) (dbmodel.OrchestrationDTO, dberr.Error) {
	condition := dbr.Eq("orchestration_id", oID)
	operation, err := r.getOrchestration(condition)
	if err != nil {
		switch {
		case dberr.IsNotFound(err):
			return dbmodel.OrchestrationDTO{}, dberr.NotFound("for ID: %s %s", oID, err)
		default:
			return dbmodel.OrchestrationDTO{}, err
		}
	}
	return operation, nil
}

func (r readSession) ListOrchestrationsByState(state string) ([]dbmodel.OrchestrationDTO, error) {
	var orchestrations []dbmodel.OrchestrationDTO

	stateCondition := dbr.Eq("state", state)

	_, err := r.session.
		Select("*").
		From(postsql.OrchestrationTableName).
		Where(stateCondition).
		Load(&orchestrations)
	if err != nil {
		return nil, dberr.Internal("Failed to get orchestrations: %s", err)
	}
	return orchestrations, nil
}

func (r readSession) ListOrchestrations() ([]dbmodel.OrchestrationDTO, dberr.Error) {
	var orchestrations []dbmodel.OrchestrationDTO

	_, err := r.session.
		Select("*").
		From(postsql.OrchestrationTableName).
		Load(&orchestrations)
	if err != nil {
		return nil, dberr.Internal("Failed to get orchestrations: %s", err)
	}
	return orchestrations, nil
}

func (r readSession) GetOperationsInProgressByType(operationType dbmodel.OperationType) ([]dbmodel.OperationDTO, dberr.Error) {
	stateCondition := dbr.Eq("state", domain.InProgress)
	typeCondition := dbr.Eq("type", operationType)
	var operations []dbmodel.OperationDTO

	_, err := r.session.
		Select("*").
		From(postsql.OperationTableName).
		Where(stateCondition).
		Where(typeCondition).
		Load(&operations)
	if err != nil {
		return nil, dberr.Internal("Failed to get operations: %s", err)
	}
	return operations, nil
}

func (r readSession) GetOperationByTypeAndInstanceID(inID string, opType dbmodel.OperationType) (dbmodel.OperationDTO, dberr.Error) {
	idCondition := dbr.Eq("instance_id", inID)
	typeCondition := dbr.Eq("type", string(opType))
	var operation dbmodel.OperationDTO

	err := r.session.
		Select("*").
		From(postsql.OperationTableName).
		Where(idCondition).
		Where(typeCondition).
		OrderDesc(postsql.CreatedAtField).
		LoadOne(&operation)

	if err != nil {
		if err == dbr.ErrNotFound {
			return dbmodel.OperationDTO{}, dberr.NotFound("cannot find operation: %s", err)
		}
		return dbmodel.OperationDTO{}, dberr.Internal("Failed to get operation: %s", err)
	}
	return operation, nil
}

func (r readSession) GetOperationsForIDs(opIDlist []string) ([]dbmodel.OperationDTO, dberr.Error) {
	var operations []dbmodel.OperationDTO

	_, err := r.session.
		Select("*").
		From(postsql.OperationTableName).
		Where("id IN ?", opIDlist).
		Load(&operations)
	if err != nil {
		return nil, dberr.Internal("Failed to get operations: %s", err)
	}
	return operations, nil
}

func (r readSession) ListRuntimeStateByRuntimeID(runtimeID string) ([]dbmodel.RuntimeStateDTO, dberr.Error) {
	stateCondition := dbr.Eq("runtime_id", runtimeID)
	var states []dbmodel.RuntimeStateDTO

	_, err := r.session.
		Select("*").
		From(postsql.RuntimeStateTableName).
		Where(stateCondition).
		Load(&states)
	if err != nil {
		return nil, dberr.Internal("Failed to get states: %s", err)
	}
	return states, nil
}

func (r readSession) getOperation(condition dbr.Builder) (dbmodel.OperationDTO, dberr.Error) {
	var operation dbmodel.OperationDTO

	err := r.session.
		Select("*").
		From(postsql.OperationTableName).
		Where(condition).
		LoadOne(&operation)

	if err != nil {
		if err == dbr.ErrNotFound {
			return dbmodel.OperationDTO{}, dberr.NotFound("cannot find operation: %s", err)
		}
		return dbmodel.OperationDTO{}, dberr.Internal("Failed to get operation: %s", err)
	}
	return operation, nil
}

func (r readSession) getOrchestration(condition dbr.Builder) (dbmodel.OrchestrationDTO, dberr.Error) {
	var operation dbmodel.OrchestrationDTO

	err := r.session.
		Select("*").
		From(postsql.OrchestrationTableName).
		Where(condition).
		LoadOne(&operation)

	if err != nil {
		if err == dbr.ErrNotFound {
			return dbmodel.OrchestrationDTO{}, dberr.NotFound("cannot find operation: %s", err)
		}
		return dbmodel.OrchestrationDTO{}, dberr.Internal("Failed to get operation: %s", err)
	}
	return operation, nil
}

func (r readSession) GetLMSTenant(name, region string) (dbmodel.LMSTenantDTO, dberr.Error) {
	var dto dbmodel.LMSTenantDTO
	err := r.session.
		Select("*").
		From(postsql.LMSTenantTableName).
		Where(dbr.Eq("name", name)).
		Where(dbr.Eq("region", region)).
		LoadOne(&dto)

	if err != nil {
		if err == dbr.ErrNotFound {
			return dbmodel.LMSTenantDTO{}, dberr.NotFound("Cannot find lms tenant for name/region: '%s/%s'", name, region)
		}
		return dbmodel.LMSTenantDTO{}, dberr.Internal("Failed to get operation: %s", err)
	}
	return dto, nil
}

func (r readSession) GetOperationStats() ([]dbmodel.OperationStatEntry, error) {
	var rows []dbmodel.OperationStatEntry
	_, err := r.session.SelectBySql(fmt.Sprintf("select type, state, count(*) as total from %s group by type, state",
		postsql.OperationTableName)).Load(&rows)
	return rows, err
}

func (r readSession) GetOperationStatsForOrchestration(orchestrationID string) ([]dbmodel.OperationStatEntry, error) {
	var rows []dbmodel.OperationStatEntry
	_, err := r.session.Select("state, count(*)").
		From(postsql.OperationTableName).
		Where("orchestration_id", orchestrationID).
		GroupBy("state").
		Load(&rows)

	return rows, err
}

func (r readSession) GetInstanceStats() ([]dbmodel.InstanceByGlobalAccountIDStatEntry, error) {
	var rows []dbmodel.InstanceByGlobalAccountIDStatEntry
	_, err := r.session.SelectBySql(fmt.Sprintf("select global_account_id, count(*) as total from %s group by global_account_id",
		postsql.InstancesTableName)).Load(&rows)
	return rows, err
}

func (r readSession) GetNumberOfInstancesForGlobalAccountID(globalAccountID string) (int, error) {
	var res struct {
		Total int
	}
	err := r.session.Select("count(*) as total").
		From(postsql.InstancesTableName).
		Where(dbr.Eq("global_account_id", globalAccountID)).
		LoadOne(&res)

	return res.Total, err
}

func (r readSession) ListInstances(limit int, page int) ([]internal.Instance, *pagination.Page, int, error) {
	var instances []internal.Instance

	order, err := pagination.ConvertPageLimitAndOrderedColumnToSQL(limit, page, postsql.CreatedAtField)
	if err != nil {
		return nil, &pagination.Page{}, -1, errors.Wrap(err, "while converting page and limit to SQL statement")
	}

	stmtWithPagination := fmt.Sprintf("SELECT * FROM %s %s", postsql.InstancesTableName, order)

	execStmt := r.session.SelectBySql(stmtWithPagination)

	_, err = execStmt.Load(&instances)
	if err != nil {
		return nil, &pagination.Page{}, -1, errors.Wrap(err, "while fetching instances")
	}

	totalCount, err := r.getInstanceCount()
	if err != nil {
		return nil, &pagination.Page{}, -1, err
	}

	hasNextPage := false

	if totalCount >= page*limit+len(instances) {
		hasNextPage = true
	}

	return instances,
		&pagination.Page{
			Count:       len(instances),
			HasNextPage: hasNextPage,
		},
		totalCount,
		nil
}

func (r readSession) getInstanceCount() (int, error) {
	var res struct {
		Total int
	}
	err := r.session.Select("count(*) as total").
		From(postsql.InstancesTableName).
		LoadOne(&res)

	return res.Total, err
}
