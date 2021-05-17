package dbsession

import (
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/persistence/dberrors"

	"github.com/gocraft/dbr/v2"
)

//go:generate mockery -name=Factory
type Factory interface {
	NewReadWriteSession() ReadWriteSession
	NewSessionWithinTransaction() (WriteSessionWithinTransaction, dberrors.Error)
}

//go:generate mockery -name=ReadSession
type ReadSession interface {
	GetProviderSpecificConfigsByProvider(provider string) ([]ProviderConfig, dberrors.Error)
}

//go:generate mockery -name=WriteSession
type WriteSession interface {
	UpdateProviderSpecificConfig(clusterID string, providerSpecificConfig string) dberrors.Error
}

//go:generate mockery -name=ReadWriteSession
type ReadWriteSession interface {
	ReadSession
	WriteSession
}

type Transaction interface {
	Commit() dberrors.Error
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

func (sf *factory) NewReadWriteSession() ReadWriteSession {
	session := sf.connection.NewSession(nil)
	return readWriteSession{
		readSession:  readSession{session: session},
		writeSession: writeSession{session: session},
	}
}

type readWriteSession struct {
	readSession
	writeSession
}

func (sf *factory) NewSessionWithinTransaction() (WriteSessionWithinTransaction, dberrors.Error) {
	dbSession := sf.connection.NewSession(nil)
	dbTransaction, err := dbSession.Begin()

	if err != nil {
		return nil, dberrors.Internal("Failed to start transaction: %s", err)
	}

	return writeSession{
		session:     dbSession,
		transaction: dbTransaction,
	}, nil
}
