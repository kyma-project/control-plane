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
	FindAllInstancesJoinedWithOperation(prct ...predicate.Predicate) ([]dbmodel.InstanceWithOperationDTO, dberr.Error)
	FindAllInstancesForRuntimes(runtimeIdList []string) ([]dbmodel.InstanceDTO, dberr.Error)
	FindAllInstancesForSubAccounts(subAccountslist []string) ([]dbmodel.InstanceDTO, dberr.Error)
	GetInstanceByID(instanceID string) (dbmodel.InstanceDTO, dberr.Error)
	GetLastOperation(instanceID string) (dbmodel.OperationDTO, dberr.Error)
	GetOperationByID(opID string) (dbmodel.OperationDTO, dberr.Error)
	GetNotFinishedOperationsByType(operationType internal.OperationType) ([]dbmodel.OperationDTO, dberr.Error)
	CountNotFinishedOperationsByInstanceID(instanceID string) (int, dberr.Error)
	GetOperationByTypeAndInstanceID(inID string, opType internal.OperationType) (dbmodel.OperationDTO, dberr.Error)
	GetOperationsByTypeAndInstanceID(inID string, opType internal.OperationType) ([]dbmodel.OperationDTO, dberr.Error)
	GetOperationsForIDs(opIdList []string) ([]dbmodel.OperationDTO, dberr.Error)
	ListOperations(filter dbmodel.OperationFilter) ([]dbmodel.OperationDTO, int, int, error)
	ListOperationsByType(operationType internal.OperationType) ([]dbmodel.OperationDTO, dberr.Error)
	GetOperationStats() ([]dbmodel.OperationStatEntry, error)
	GetInstanceStats() ([]dbmodel.InstanceByGlobalAccountIDStatEntry, error)
	GetNumberOfInstancesForGlobalAccountID(globalAccountID string) (int, error)
	GetRuntimeStateByOperationID(operationID string) (dbmodel.RuntimeStateDTO, dberr.Error)
	ListRuntimeStateByRuntimeID(runtimeID string) ([]dbmodel.RuntimeStateDTO, dberr.Error)
	GetOrchestrationByID(oID string) (dbmodel.OrchestrationDTO, dberr.Error)
	ListOrchestrations(filter dbmodel.OrchestrationFilter) ([]dbmodel.OrchestrationDTO, int, int, error)
	ListInstances(filter dbmodel.InstanceFilter) ([]dbmodel.InstanceDTO, int, int, error)
	ListOperationsByOrchestrationID(orchestrationID string, filter dbmodel.OperationFilter) ([]dbmodel.OperationDTO, int, int, error)
	GetOperationStatsForOrchestration(orchestrationID string) ([]dbmodel.OperationStatEntry, error)
	GetLatestRuntimeStateByRuntimeID(runtimeID string) (dbmodel.RuntimeStateDTO, dberr.Error)
	GetLatestRuntimeStateWithReconcilerInputByRuntimeID(runtimeID string) (dbmodel.RuntimeStateDTO, dberr.Error)
	GetLatestRuntimeStateWithKymaVersionByRuntimeID(runtimeID string) (dbmodel.RuntimeStateDTO, dberr.Error)
}

//go:generate mockery -name=WriteSession
type WriteSession interface {
	InsertInstance(instance dbmodel.InstanceDTO) dberr.Error
	UpdateInstance(instance dbmodel.InstanceDTO) dberr.Error
	DeleteInstance(instanceID string) dberr.Error
	InsertOperation(dto dbmodel.OperationDTO) dberr.Error
	UpdateOperation(dto dbmodel.OperationDTO) dberr.Error
	InsertOrchestration(o dbmodel.OrchestrationDTO) dberr.Error
	UpdateOrchestration(o dbmodel.OrchestrationDTO) dberr.Error
	InsertRuntimeState(state dbmodel.RuntimeStateDTO) dberr.Error
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
