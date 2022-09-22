package events

import (
	"fmt"
	"log"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
	"github.com/sirupsen/logrus"
)

var ev *events

type events struct {
	postsql.Factory

	log logrus.FieldLogger
}

func New(sess postsql.Factory, log logrus.FieldLogger) *events {
	if ev == nil {
		ev = &events{
			Factory: sess,
			log:     log,
		}
		go ev.gc()
	}
	return ev
}

func Infof(instanceID, operationID, format string, args ...string) {
	ev.InsertEvent(dbmodel.InfoEventLevel, fmt.Sprintf(format, args), instanceID, operationID)
}

func Errorf(instanceID, operationID string, err error, format string, args ...string) {
	ev.InsertEvent(dbmodel.ErrorEventLevel, fmt.Sprintf("%v: %v", fmt.Sprintf(format, args), err), instanceID, operationID)
}

func (e *events) InsertEvent(eventLevel dbmodel.EventLevel, message, instanceID, operationID string) {
	if e != nil {
		sess := e.NewWriteSession()
		if err := sess.InsertEvent(eventLevel, message, instanceID, operationID); err != nil {
			e.log.Errorf("failed to insert event [%v] %v/%v %q: %v", eventLevel, instanceID, operationID, message, err)
		}
	} else {
		log.Printf("no event sink set for event [%v] %v/%v %q\n", eventLevel, instanceID, operationID, message)
	}
}

func (e *events) gc() {
	// TODO: implement events garbace collection with retention policy from config
	for {
	}
}
