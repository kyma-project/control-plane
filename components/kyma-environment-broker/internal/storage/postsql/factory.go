package postsql

import (
	dbr "github.com/gocraft/dbr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/predicate"
)

//go:generate mockery -name=Factory
type Factory interface {
	NewReadSession() ReadSession
	NewWriteSession() WriteSession
	NewSessionWithinTransaction() (WriteSessionWithinTransaction, dberr.Error)
}

//go:generate mockery -name=ReadSession
type ReadSession interface {
	FindAllInstancesJoinedWithOperation(prct ...predicate.Predicate) ([]internal.InstanceWithOperation, dberr.Error)
	FindAllInstancesForRuntimes(runtimeIdList []string) ([]internal.Instance, dberr.Error)
	FindAllInstancesForSubAccounts(subAccountslist []string) ([]internal.Instance, dberr.Error)
	GetInstanceByID(instanceID string) (internal.Instance, dberr.Error)
	GetLastOperation(instanceID string) (dbmodel.OperationDTO, dberr.Error)
	GetOperationByID(opID string) (dbmodel.OperationDTO, dberr.Error)
	GetOperationsInProgressByType(operationType dbmodel.OperationType) ([]dbmodel.OperationDTO, dberr.Error)
	GetOperationByTypeAndInstanceID(inID string, opType dbmodel.OperationType) (dbmodel.OperationDTO, dberr.Error)
	GetOperationsByTypeAndInstanceID(inID string, opType dbmodel.OperationType) ([]dbmodel.OperationDTO, dberr.Error)
	GetOperationsForIDs(opIdList []string) ([]dbmodel.OperationDTO, dberr.Error)
	ListOperations(filter dbmodel.OperationFilter) ([]dbmodel.OperationDTO, int, int, error)
	ListOperationsByType(operationType dbmodel.OperationType) ([]dbmodel.OperationDTO, dberr.Error)
	GetLMSTenant(name, region string) (dbmodel.LMSTenantDTO, dberr.Error)
	GetOperationStats() ([]dbmodel.OperationStatEntry, error)
	GetInstanceStats() ([]dbmodel.InstanceByGlobalAccountIDStatEntry, error)
	GetNumberOfInstancesForGlobalAccountID(globalAccountID string) (int, error)
	GetRuntimeStateByOperationID(operationID string) (dbmodel.RuntimeStateDTO, dberr.Error)
	ListRuntimeStateByRuntimeID(runtimeID string) ([]dbmodel.RuntimeStateDTO, dberr.Error)
	GetOrchestrationByID(oID string) (dbmodel.OrchestrationDTO, dberr.Error)
	ListOrchestrations(filter dbmodel.OrchestrationFilter) ([]dbmodel.OrchestrationDTO, int, int, error)
	ListInstances(filter dbmodel.InstanceFilter) ([]internal.Instance, int, int, error)
	ListOperationsByOrchestrationID(orchestrationID string, filter dbmodel.OperationFilter) ([]dbmodel.OperationDTO, int, int, error)
	GetOperationStatsForOrchestration(orchestrationID string) ([]dbmodel.OperationStatEntry, error)
}

//go:generate mockery -name=WriteSession
type WriteSession interface {
	InsertInstance(instance internal.Instance) dberr.Error
	UpdateInstance(instance internal.Instance) dberr.Error
	DeleteInstance(instanceID string) dberr.Error
	InsertOperation(dto dbmodel.OperationDTO) dberr.Error
	UpdateOperation(dto dbmodel.OperationDTO) dberr.Error
	InsertOrchestration(o dbmodel.OrchestrationDTO) dberr.Error
	UpdateOrchestration(o dbmodel.OrchestrationDTO) dberr.Error
	InsertRuntimeState(state dbmodel.RuntimeStateDTO) dberr.Error
	InsertLMSTenant(dto dbmodel.LMSTenantDTO) dberr.Error
}

type Transaction interface {
	Commit() dberr.Error
	RollbackUnlessCommitted()
}

//go:generate mockery -name=WriteSessionWithinTransaction
type WriteSessionWithinTransaction interface {
	WriteSession
	Transaction
}

type factory struct {
	connection *dbr.Connection
}

func NewFactory(connection *dbr.Connection) Factory {
	return &factory{
		connection: connection,
	}
}

func (sf *factory) NewReadSession() ReadSession {
	return readSession{
		session: sf.connection.NewSession(nil),
	}
}

func (sf *factory) NewWriteSession() WriteSession {
	return writeSession{
		session: sf.connection.NewSession(nil),
	}
}

func (sf *factory) NewSessionWithinTransaction() (WriteSessionWithinTransaction, dberr.Error) {
	dbSession := sf.connection.NewSession(nil)
	dbTransaction, err := dbSession.Begin()

	if err != nil {
		return nil, dberr.Internal("Failed to start transaction: %s", err)
	}

	return writeSession{
		session:     dbSession,
		transaction: dbTransaction,
	}, nil
}
