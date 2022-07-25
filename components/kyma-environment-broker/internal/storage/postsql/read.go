package postsql

import (
	"fmt"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/pkg/errors"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/predicate"

	"github.com/gocraft/dbr"
	"github.com/pivotal-cf/brokerapi/v8/domain"
)

type readSession struct {
	session *dbr.Session
}

func (r readSession) getInstancesJoinedWithOperationStatement() *dbr.SelectStmt {
	join := fmt.Sprintf("%s.instance_id = %s.instance_id", InstancesTableName, OperationTableName)
	stmt := r.session.
		Select("instances.instance_id, instances.runtime_id, instances.global_account_id, instances.subscription_global_account_id, instances.service_id,"+
			" instances.service_plan_id, instances.dashboard_url, instances.provisioning_parameters, instances.created_at,"+
			" instances.updated_at, instances.deleted_at, instances.sub_account_id, instances.service_name, instances.service_plan_name,"+
			" instances.provider_region, instances.provider, operations.state, operations.description, operations.type, operations.created_at AS operation_created_at, operations.data").
		From(InstancesTableName).
		LeftJoin(OperationTableName, join)
	return stmt
}

func (r readSession) FindAllInstancesJoinedWithOperation(prct ...predicate.Predicate) ([]dbmodel.InstanceWithOperationDTO, dberr.Error) {
	var instances []dbmodel.InstanceWithOperationDTO

	stmt := r.getInstancesJoinedWithOperationStatement()
	for _, p := range prct {
		p.ApplyToPostgres(stmt)
	}

	if _, err := stmt.Load(&instances); err != nil {
		return nil, dberr.Internal("Failed to fetch all instances: %s", err)
	}

	return instances, nil
}

func (r readSession) GetInstanceByID(instanceID string) (dbmodel.InstanceDTO, dberr.Error) {
	var instance dbmodel.InstanceDTO

	err := r.session.
		Select("*").
		From(InstancesTableName).
		Where(dbr.Eq("instance_id", instanceID)).
		LoadOne(&instance)

	if err != nil {
		if err == dbr.ErrNotFound {
			return dbmodel.InstanceDTO{}, dberr.NotFound("Cannot find Instance for instanceID:'%s'", instanceID)
		}
		return dbmodel.InstanceDTO{}, dberr.Internal("Failed to get Instance: %s", err)
	}

	return instance, nil
}

func (r readSession) FindAllInstancesForRuntimes(runtimeIdList []string) ([]dbmodel.InstanceDTO, dberr.Error) {
	var instances []dbmodel.InstanceDTO

	err := r.session.
		Select("*").
		From(InstancesTableName).
		Where("runtime_id IN ?", runtimeIdList).
		LoadOne(&instances)

	if err != nil {
		if err == dbr.ErrNotFound {
			return []dbmodel.InstanceDTO{}, dberr.NotFound("Cannot find Instances for runtime ID list: '%v'", runtimeIdList)
		}
		return []dbmodel.InstanceDTO{}, dberr.Internal("Failed to get Instances: %s", err)
	}
	return instances, nil
}

func (r readSession) FindAllInstancesForSubAccounts(subAccountslist []string) ([]dbmodel.InstanceDTO, dberr.Error) {
	var instances []dbmodel.InstanceDTO

	err := r.session.
		Select("*").
		From(InstancesTableName).
		Where("sub_account_id IN ?", subAccountslist).
		LoadOne(&instances)

	if err != nil {
		if err == dbr.ErrNotFound {
			return []dbmodel.InstanceDTO{}, nil
		}
		return []dbmodel.InstanceDTO{}, dberr.Internal("Failed to get Instances: %s", err)
	}
	return instances, nil
}

func (r readSession) GetLastOperation(instanceID string) (dbmodel.OperationDTO, dberr.Error) {
	inst := dbr.Eq("instance_id", instanceID)
	state := dbr.Neq("state", []string{orchestration.Pending, orchestration.Canceled})
	condition := dbr.And(inst, state)
	operation, err := r.getLastOperation(condition)
	if err != nil {
		switch {
		case dberr.IsNotFound(err):
			return dbmodel.OperationDTO{}, dberr.NotFound("for instance ID: %s %s", instanceID, err)
		default:
			return dbmodel.OperationDTO{}, err
		}
	}
	return operation, nil
}

func (r readSession) GetOperationByInstanceID(instanceId string) (dbmodel.OperationDTO, dberr.Error) {
	condition := dbr.Eq("instance_id", instanceId)
	operation, err := r.getOperation(condition)
	if err != nil {
		switch {
		case dberr.IsNotFound(err):
			return dbmodel.OperationDTO{}, dberr.NotFound("for instance_id: %s %s", instanceId, err)
		default:
			return dbmodel.OperationDTO{}, err
		}
	}
	return operation, nil
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

func (r readSession) ListOperations(filter dbmodel.OperationFilter) ([]dbmodel.OperationDTO, int, int, error) {
	var operations []dbmodel.OperationDTO

	stmt := r.session.Select("*").
		From(OperationTableName).
		OrderBy(CreatedAtField)

	// Add pagination if provided
	if filter.Page > 0 && filter.PageSize > 0 {
		stmt.Paginate(uint64(filter.Page), uint64(filter.PageSize))
	}

	// Apply filtering if provided
	addOperationFilters(stmt, filter)

	_, err := stmt.Load(&operations)

	totalCount, err := r.getOperationCount(filter)
	if err != nil {
		return nil, -1, -1, err
	}

	return operations,
		len(operations),
		totalCount,
		nil
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

func (r readSession) ListOrchestrations(filter dbmodel.OrchestrationFilter) ([]dbmodel.OrchestrationDTO, int, int, error) {
	var orchestrations []dbmodel.OrchestrationDTO

	stmt := r.session.Select("*").
		From(OrchestrationTableName).
		OrderBy(CreatedAtField)

	// Add pagination if provided
	if filter.Page > 0 && filter.PageSize > 0 {
		stmt.Paginate(uint64(filter.Page), uint64(filter.PageSize))
	}

	// Apply filtering if provided
	addOrchestrationFilters(stmt, filter)

	_, err := stmt.Load(&orchestrations)

	totalCount, err := r.getOrchestrationCount(filter)
	if err != nil {
		return nil, -1, -1, err
	}

	return orchestrations,
		len(orchestrations),
		totalCount,
		nil
}

func (r readSession) CountNotFinishedOperationsByInstanceID(instanceID string) (int, dberr.Error) {
	stateInProgress := dbr.Eq("state", domain.InProgress)
	statePending := dbr.Eq("state", orchestration.Pending)
	stateCondition := dbr.Or(statePending, stateInProgress)
	instanceIDCondition := dbr.Eq("instance_id", instanceID)

	var res struct {
		Total int
	}
	err := r.session.Select("count(*) as total").
		From(OperationTableName).
		Where(stateCondition).
		Where(instanceIDCondition).
		LoadOne(&res)

	if err != nil {
		return 0, dberr.Internal("Failed to count operations: %s", err)
	}
	return res.Total, nil
}

func (r readSession) GetNotFinishedOperationsByType(operationType internal.OperationType) ([]dbmodel.OperationDTO, dberr.Error) {
	stateInProgress := dbr.Eq("state", domain.InProgress)
	statePending := dbr.Eq("state", orchestration.Pending)
	stateCondition := dbr.Or(statePending, stateInProgress)
	typeCondition := dbr.Eq("type", operationType)
	var operations []dbmodel.OperationDTO

	_, err := r.session.
		Select("*").
		From(OperationTableName).
		Where(stateCondition).
		Where(typeCondition).
		Load(&operations)
	if err != nil {
		return nil, dberr.Internal("Failed to get operations: %s", err)
	}
	return operations, nil
}

func (r readSession) GetOperationByTypeAndInstanceID(inID string, opType internal.OperationType) (dbmodel.OperationDTO, dberr.Error) {
	idCondition := dbr.Eq("instance_id", inID)
	typeCondition := dbr.Eq("type", string(opType))
	var operation dbmodel.OperationDTO

	err := r.session.
		Select("*").
		From(OperationTableName).
		Where(idCondition).
		Where(typeCondition).
		OrderDesc(CreatedAtField).
		LoadOne(&operation)

	if err != nil {
		if err == dbr.ErrNotFound {
			return dbmodel.OperationDTO{}, dberr.NotFound("cannot find operation: %s", err)
		}
		return dbmodel.OperationDTO{}, dberr.Internal("Failed to get operation: %s", err)
	}
	return operation, nil
}

func (r readSession) GetOperationsByTypeAndInstanceID(inID string, opType internal.OperationType) ([]dbmodel.OperationDTO, dberr.Error) {
	idCondition := dbr.Eq("instance_id", inID)
	typeCondition := dbr.Eq("type", string(opType))
	var operations []dbmodel.OperationDTO

	_, err := r.session.
		Select("*").
		From(OperationTableName).
		Where(idCondition).
		Where(typeCondition).
		OrderDesc(CreatedAtField).
		Load(&operations)

	if err != nil {
		return []dbmodel.OperationDTO{}, dberr.Internal("Failed to get operations: %s", err)
	}
	return operations, nil
}

func (r readSession) GetOperationsByInstanceID(inID string) ([]dbmodel.OperationDTO, dberr.Error) {
	idCondition := dbr.Eq("instance_id", inID)
	var operations []dbmodel.OperationDTO

	_, err := r.session.
		Select("*").
		From(OperationTableName).
		Where(idCondition).
		OrderDesc(CreatedAtField).
		Load(&operations)

	if err != nil {
		return []dbmodel.OperationDTO{}, dberr.Internal("Failed to get operations: %s", err)
	}
	return operations, nil
}

func (r readSession) GetOperationsForIDs(opIDlist []string) ([]dbmodel.OperationDTO, dberr.Error) {
	var operations []dbmodel.OperationDTO

	_, err := r.session.
		Select("*").
		From(OperationTableName).
		Where("id IN ?", opIDlist).
		Load(&operations)
	if err != nil {
		return nil, dberr.Internal("Failed to get operations: %s", err)
	}
	return operations, nil
}

func (r readSession) ListOperationsByType(operationType internal.OperationType) ([]dbmodel.OperationDTO, dberr.Error) {
	typeCondition := dbr.Eq("type", operationType)
	var operations []dbmodel.OperationDTO

	_, err := r.session.
		Select("*").
		From(OperationTableName).
		Where(typeCondition).
		Load(&operations)
	if err != nil {
		return nil, dberr.Internal("Failed to get operations: %s", err)
	}
	return operations, nil
}

func (r readSession) ListOperationsByOrchestrationID(orchestrationID string, filter dbmodel.OperationFilter) ([]dbmodel.OperationDTO, int, int, error) {
	var ops []dbmodel.OperationDTO
	condition := dbr.Eq("orchestration_id", orchestrationID)

	stmt := r.session.
		Select("*").
		From(OperationTableName).
		Where(condition).
		OrderBy(CreatedAtField)

	// Add pagination if provided
	if filter.Page > 0 && filter.PageSize > 0 {
		stmt.Paginate(uint64(filter.Page), uint64(filter.PageSize))
	}

	// Apply filtering if provided
	addOperationFilters(stmt, filter)

	_, err := stmt.Load(&ops)
	if err != nil {
		return nil, -1, -1, dberr.Internal("Failed to get operations: %s", err)
	}

	totalCount, err := r.getUpgradeOperationCount(orchestrationID, filter)
	if err != nil {
		return nil, -1, -1, err
	}

	return ops,
		len(ops),
		totalCount,
		nil
}

func (r readSession) GetRuntimeStateByOperationID(operationID string) (dbmodel.RuntimeStateDTO, dberr.Error) {
	var state dbmodel.RuntimeStateDTO

	err := r.session.
		Select("*").
		From(RuntimeStateTableName).
		Where(dbr.Eq("operation_id", operationID)).
		LoadOne(&state)

	if err != nil {
		if err == dbr.ErrNotFound {
			return dbmodel.RuntimeStateDTO{}, dberr.NotFound("cannot find runtime state: %s", err)
		}
		return dbmodel.RuntimeStateDTO{}, dberr.Internal("Failed to get runtime state: %s", err)
	}
	return state, nil
}

func (r readSession) ListRuntimeStateByRuntimeID(runtimeID string) ([]dbmodel.RuntimeStateDTO, dberr.Error) {
	stateCondition := dbr.Eq("runtime_id", runtimeID)
	var states []dbmodel.RuntimeStateDTO

	_, err := r.session.
		Select("*").
		From(RuntimeStateTableName).
		Where(stateCondition).
		OrderDesc(CreatedAtField).
		Load(&states)
	if err != nil {
		return nil, dberr.Internal("Failed to get states: %s", err)
	}
	return states, nil
}

func (r readSession) GetLatestRuntimeStateByRuntimeID(runtimeID string) (dbmodel.RuntimeStateDTO, dberr.Error) {
	var state dbmodel.RuntimeStateDTO

	count, err := r.session.
		Select("*").
		From(RuntimeStateTableName).
		Where(dbr.Eq("runtime_id", runtimeID)).
		OrderDesc(CreatedAtField).
		Limit(1).
		Load(&state)
	if err != nil {
		if err == dbr.ErrNotFound {
			return dbmodel.RuntimeStateDTO{}, dberr.NotFound("cannot find runtime state: %s", err)
		}
		return dbmodel.RuntimeStateDTO{}, dberr.Internal("Failed to get the latest runtime state: %s", err)
	}
	if count == 0 {
		return dbmodel.RuntimeStateDTO{}, dberr.NotFound("cannot find runtime state: %s", err)
	}
	return state, nil
}

func (r readSession) GetLatestRuntimeStateWithReconcilerInputByRuntimeID(runtimeID string) (dbmodel.RuntimeStateDTO, dberr.Error) {
	var state dbmodel.RuntimeStateDTO
	runtimeIDIsEqual := dbr.Eq("runtime_id", runtimeID)
	reconcilerInputIsNotEmptyString := dbr.Neq("cluster_setup", "")
	reconcilerInputIsNotNil := dbr.Neq("cluster_setup", nil)
	innerCondition := dbr.And(reconcilerInputIsNotEmptyString, reconcilerInputIsNotNil)
	condition := dbr.And(runtimeIDIsEqual, innerCondition)

	count, err := r.session.
		Select("*").
		From(RuntimeStateTableName).
		Where(condition).
		OrderDesc(CreatedAtField).
		Limit(1).
		Load(&state)
	if err != nil {
		if err == dbr.ErrNotFound {
			return dbmodel.RuntimeStateDTO{}, dberr.NotFound("cannot find runtime state: %s", err)
		}
		return dbmodel.RuntimeStateDTO{}, dberr.Internal("Failed to get the latest runtime state with reconciler input: %s", err)
	}
	if count == 0 {
		return dbmodel.RuntimeStateDTO{}, dberr.NotFound("cannot find runtime state with reconciler input: %s", err)
	}
	return state, nil
}

func (r readSession) GetLatestRuntimeStateWithKymaVersionByRuntimeID(runtimeID string) (dbmodel.RuntimeStateDTO, dberr.Error) {
	var state dbmodel.RuntimeStateDTO
	condition := dbr.And(dbr.Eq("runtime_id", runtimeID),
		dbr.And(dbr.Neq("kyma_version", nil), dbr.Neq("kyma_version", "")),
	)

	count, err := r.session.
		Select("*").
		From(RuntimeStateTableName).
		Where(condition).
		OrderDesc(CreatedAtField).
		Limit(1).
		Load(&state)
	if err != nil {
		if err == dbr.ErrNotFound {
			return state, dberr.NotFound("cannot find latest runtime state with kyma version: %s", err)
		}
		return state, dberr.Internal("Failed to get the latest runtime state with kyma version: %s", err)
	}
	if count == 0 {
		return state, dberr.NotFound("found 0 latest runtime states with kyma version: %s", err)
	}
	return state, nil
}

func (r readSession) GetLatestRuntimeStateWithOIDCConfigByRuntimeID(runtimeID string) (dbmodel.RuntimeStateDTO, dberr.Error) {
	var state dbmodel.RuntimeStateDTO
	condition := dbr.And(dbr.Eq("runtime_id", runtimeID),
		dbr.Expr("cluster_config::json->>'oidcConfig' != ?", "null"),
	)

	count, err := r.session.
		Select("*").
		From(RuntimeStateTableName).
		Where(condition).
		OrderDesc(CreatedAtField).
		Limit(1).
		Load(&state)
	if err != nil {
		if err == dbr.ErrNotFound {
			return state, dberr.NotFound("cannot find latest runtime state with OIDC config: %s", err)
		}
		return state, dberr.Internal("failed to get the latest runtime state with OIDC config: %s", err)
	}
	if count == 0 {
		return state, dberr.NotFound("found 0 latest runtime states with OIDC config: %s", err)
	}
	return state, nil
}

func (r readSession) getOperation(condition dbr.Builder) (dbmodel.OperationDTO, dberr.Error) {
	var operation dbmodel.OperationDTO

	err := r.session.
		Select("*").
		From(OperationTableName).
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

func (r readSession) getLastOperation(condition dbr.Builder) (dbmodel.OperationDTO, dberr.Error) {
	var operation dbmodel.OperationDTO

	count, err := r.session.
		Select("*").
		From(OperationTableName).
		Where(condition).
		OrderDesc(CreatedAtField).
		Limit(1).
		Load(&operation)
	if err != nil {
		if err == dbr.ErrNotFound {
			return dbmodel.OperationDTO{}, dberr.NotFound("cannot find operation: %s", err)
		}
		return dbmodel.OperationDTO{}, dberr.Internal("Failed to get operation: %s", err)
	}
	if count == 0 {
		return dbmodel.OperationDTO{}, dberr.NotFound("cannot find operation: %s", err)
	}

	return operation, nil
}

func (r readSession) getOrchestration(condition dbr.Builder) (dbmodel.OrchestrationDTO, dberr.Error) {
	var operation dbmodel.OrchestrationDTO

	err := r.session.
		Select("*").
		From(OrchestrationTableName).
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

func (r readSession) GetOperationStats() ([]dbmodel.OperationStatEntry, error) {
	var rows []dbmodel.OperationStatEntry
	_, err := r.session.SelectBySql(fmt.Sprintf("select type, state, provisioning_parameters ->> 'plan_id' AS plan_id from %s",
		OperationTableName)).Load(&rows)
	return rows, err
}

func (r readSession) GetOperationStatsForOrchestration(orchestrationID string) ([]dbmodel.OperationStatEntry, error) {
	var rows []dbmodel.OperationStatEntry
	_, err := r.session.SelectBySql(fmt.Sprintf("select type, state, provisioning_parameters ->> 'plan_id' AS plan_id from %s where orchestration_id='%s'",
		OperationTableName, orchestrationID)).Load(&rows)
	return rows, err
}

func (r readSession) GetInstanceStats() ([]dbmodel.InstanceByGlobalAccountIDStatEntry, error) {
	var rows []dbmodel.InstanceByGlobalAccountIDStatEntry
	_, err := r.session.SelectBySql(fmt.Sprintf("select global_account_id, count(*) as total from %s group by global_account_id",
		InstancesTableName)).Load(&rows)
	return rows, err
}

func (r readSession) GetERSContextStats() ([]dbmodel.InstanceERSContextStatsEntry, error) {
	var rows []dbmodel.InstanceERSContextStatsEntry
	// group existing instances by license_Type from the last operation that is not pending or canceled
	_, err := r.session.SelectBySql(`
SELECT license_type, count(1) as total
FROM (
    SELECT DISTINCT ON (instances.instance_id) instances.instance_id, operations.id, state, type, (operations.provisioning_parameters->'ers_context'->'license_type')::VARCHAR AS license_type
    FROM operations
    INNER JOIN instances
    ON operations.instance_id = instances.instance_id
    WHERE operations.state != 'pending' OR operations.state != 'canceled'
    ORDER BY instance_id, operations.created_at DESC
) t
GROUP BY license_type;
`).Load(&rows)
	return rows, err
}

func (r readSession) GetNumberOfInstancesForGlobalAccountID(globalAccountID string) (int, error) {
	var res struct {
		Total int
	}
	err := r.session.Select("count(*) as total").
		From(InstancesTableName).
		Where(dbr.Eq("global_account_id", globalAccountID)).
		LoadOne(&res)

	return res.Total, err
}

func (r readSession) ListInstances(filter dbmodel.InstanceFilter) ([]dbmodel.InstanceDTO, int, int, error) {
	var instances []dbmodel.InstanceDTO

	// Base select and order by created at
	var stmt *dbr.SelectStmt
	// Find and join the last operation for each instance matching the state filter(s).
	// Last operation is found with the greatest-n-per-group problem solved with OUTER JOIN, followed by a (INNER) JOIN to get instance columns.
	stmt = r.session.
		Select(fmt.Sprintf("%s.*", InstancesTableName)).
		From(InstancesTableName).
		Join(dbr.I(OperationTableName).As("o1"), fmt.Sprintf("%s.instance_id = o1.instance_id", InstancesTableName)).
		LeftJoin(dbr.I(OperationTableName).As("o2"), fmt.Sprintf("%s.instance_id = o2.instance_id AND o1.created_at < o2.created_at AND o2.state NOT IN ('%s', '%s')", InstancesTableName, orchestration.Pending, orchestration.Canceled)).
		Where("o2.created_at IS NULL").
		Where(fmt.Sprintf("o1.state NOT IN ('%s', '%s')", orchestration.Pending, orchestration.Canceled)).
		OrderBy(fmt.Sprintf("%s.%s", InstancesTableName, CreatedAtField))

	if len(filter.States) > 0 {
		stateFilters := buildInstanceStateFilters("o1", filter)
		stmt.Where(stateFilters)
	}

	// Add pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		stmt = stmt.Paginate(uint64(filter.Page), uint64(filter.PageSize))
	}

	addInstanceFilters(stmt, filter)

	_, err := stmt.Load(&instances)
	if err != nil {
		return nil, -1, -1, errors.Wrap(err, "while fetching instances")
	}

	totalCount, err := r.getInstanceCount(filter)
	if err != nil {
		return nil, -1, -1, err
	}

	return instances,
		len(instances),
		totalCount,
		nil
}

func (r readSession) getInstanceCount(filter dbmodel.InstanceFilter) (int, error) {
	var res struct {
		Total int
	}
	var stmt *dbr.SelectStmt
	stmt = r.session.
		Select("count(*) as total").
		From(InstancesTableName).
		Join(dbr.I(OperationTableName).As("o1"), fmt.Sprintf("%s.instance_id = o1.instance_id", InstancesTableName)).
		LeftJoin(dbr.I(OperationTableName).As("o2"), fmt.Sprintf("%s.instance_id = o2.instance_id AND o1.created_at < o2.created_at AND o2.state NOT IN ('%s', '%s')", InstancesTableName, orchestration.Pending, orchestration.Canceled)).
		Where("o2.created_at IS NULL").
		Where(fmt.Sprintf("o1.state NOT IN ('%s', '%s')", orchestration.Pending, orchestration.Canceled))

	if len(filter.States) > 0 {
		stateFilters := buildInstanceStateFilters("o1", filter)
		stmt.Where(stateFilters)
	}

	addInstanceFilters(stmt, filter)
	err := stmt.LoadOne(&res)

	return res.Total, err
}

func buildInstanceStateFilters(table string, filter dbmodel.InstanceFilter) dbr.Builder {
	var exprs []dbr.Builder
	for _, s := range filter.States {
		switch s {
		case dbmodel.InstanceSucceeded:
			exprs = append(exprs, dbr.And(
				dbr.Eq(fmt.Sprintf("%s.state", table), domain.Succeeded),
				dbr.Neq(fmt.Sprintf("%s.type", table), internal.OperationTypeDeprovision),
			))
		case dbmodel.InstanceFailed:
			exprs = append(exprs, dbr.And(
				dbr.Or(
					dbr.Eq(fmt.Sprintf("%s.type", table), internal.OperationTypeProvision),
					dbr.Eq(fmt.Sprintf("%s.type", table), internal.OperationTypeDeprovision),
				),
				dbr.Eq(fmt.Sprintf("%s.state", table), domain.Failed),
			))
		case dbmodel.InstanceError:
			exprs = append(exprs, dbr.And(
				dbr.Neq(fmt.Sprintf("%s.type", table), internal.OperationTypeProvision),
				dbr.Neq(fmt.Sprintf("%s.type", table), internal.OperationTypeDeprovision),
				dbr.Eq(fmt.Sprintf("%s.state", table), domain.Failed),
			))
		case dbmodel.InstanceProvisioning:
			exprs = append(exprs, dbr.And(
				dbr.Eq(fmt.Sprintf("%s.type", table), internal.OperationTypeProvision),
				dbr.Eq(fmt.Sprintf("%s.state", table), domain.InProgress),
			))
		case dbmodel.InstanceDeprovisioning:
			exprs = append(exprs, dbr.And(
				dbr.Eq(fmt.Sprintf("%s.type", table), internal.OperationTypeDeprovision),
				dbr.Eq(fmt.Sprintf("%s.state", table), domain.InProgress),
			))
		case dbmodel.InstanceUpgrading:
			exprs = append(exprs, dbr.And(
				dbr.Like(fmt.Sprintf("%s.type", table), "upgrade%"),
				dbr.Eq(fmt.Sprintf("%s.state", table), domain.InProgress),
			))
		case dbmodel.InstanceUpdating:
			exprs = append(exprs, dbr.And(
				dbr.Eq(fmt.Sprintf("%s.type", table), internal.OperationTypeUpdate),
				dbr.Eq(fmt.Sprintf("%s.state", table), domain.InProgress),
			))
		case dbmodel.InstanceDeprovisioned:
			exprs = append(exprs, dbr.And(
				dbr.Eq(fmt.Sprintf("%s.type", table), internal.OperationTypeDeprovision),
				dbr.Eq(fmt.Sprintf("%s.state", table), domain.Succeeded),
			))
		case dbmodel.InstanceNotDeprovisioned:
			exprs = append(exprs, dbr.Or(
				dbr.Neq(fmt.Sprintf("%s.type", table), internal.OperationTypeDeprovision),
				dbr.Neq(fmt.Sprintf("%s.state", table), domain.Succeeded),
			))
		}
	}

	return dbr.Or(exprs...)
}

func addInstanceFilters(stmt *dbr.SelectStmt, filter dbmodel.InstanceFilter) {
	if len(filter.GlobalAccountIDs) > 0 {
		stmt.Where("instances.global_account_id IN ?", filter.GlobalAccountIDs)
	}
	if len(filter.SubAccountIDs) > 0 {
		stmt.Where("instances.sub_account_id IN ?", filter.SubAccountIDs)
	}
	if len(filter.InstanceIDs) > 0 {
		stmt.Where("instances.instance_id IN ?", filter.InstanceIDs)
	}
	if len(filter.RuntimeIDs) > 0 {
		stmt.Where("instances.runtime_id IN ?", filter.RuntimeIDs)
	}
	if len(filter.Regions) > 0 {
		stmt.Where("instances.provider_region IN ?", filter.Regions)
	}
	if len(filter.Plans) > 0 {
		stmt.Where("instances.service_plan_name IN ?", filter.Plans)
	}
	if len(filter.Shoots) > 0 {
		shootNameMatch := fmt.Sprintf(`^(%s)$`, strings.Join(filter.Shoots, "|"))
		stmt.Where("o1.data::json->>'shoot_name' ~ ?", shootNameMatch)
	}
}

func addOrchestrationFilters(stmt *dbr.SelectStmt, filter dbmodel.OrchestrationFilter) {
	if len(filter.Types) > 0 {
		stmt.Where("type IN ?", filter.Types)
	}
	if len(filter.States) > 0 {
		stmt.Where("state IN ?", filter.States)
	}
}

func addOperationFilters(stmt *dbr.SelectStmt, filter dbmodel.OperationFilter) {
	if len(filter.States) > 0 {
		stmt.Where("state IN ?", filter.States)
	}
}

func (r readSession) getOperationCount(filter dbmodel.OperationFilter) (int, error) {
	var res struct {
		Total int
	}
	stmt := r.session.Select("count(*) as total").
		From(OperationTableName)
	addOperationFilters(stmt, filter)
	err := stmt.LoadOne(&res)

	return res.Total, err
}

func (r readSession) getUpgradeOperationCount(orchestrationID string, filter dbmodel.OperationFilter) (int, error) {
	var res struct {
		Total int
	}
	stmt := r.session.Select("count(*) as total").
		From(OperationTableName).
		Where(dbr.Eq("orchestration_id", orchestrationID))
	addOperationFilters(stmt, filter)
	err := stmt.LoadOne(&res)

	return res.Total, err
}

func (r readSession) getOrchestrationCount(filter dbmodel.OrchestrationFilter) (int, error) {
	var res struct {
		Total int
	}
	stmt := r.session.Select("count(*) as total").From(OrchestrationTableName)
	addOrchestrationFilters(stmt, filter)
	err := stmt.LoadOne(&res)

	return res.Total, err
}
