package postsql

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
)

type events struct {
	postsql.Factory
}

func NewEvents(sess postsql.Factory) *events {
	return &events{
		Factory: sess,
	}
}

func (e *events) Info(message, instanceID, operationID string) error {
	sess := e.NewWriteSession()
	return sess.InsertEvent(dbmodel.InfoEventLevel, message, instanceID, operationID)
}

func (e *events) Error(err error, instanceID, operationID string) error {
	sess := e.NewWriteSession()
	return sess.InsertEvent(dbmodel.ErrorEventLevel, message.Error(), instanceID, operationID)
}
