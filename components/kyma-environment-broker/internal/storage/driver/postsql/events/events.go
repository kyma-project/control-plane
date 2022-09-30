package events

import (
	"fmt"
	"log"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Retention     time.Duration `envconfig:"default=72h"`
	PollingPeriod time.Duration `envconfig:"default=1h"`
}

var ev *events

type events struct {
	postsql.Factory

	log logrus.FieldLogger
}

func New(cfg Config, sess postsql.Factory, log logrus.FieldLogger) *events {
	if ev == nil {
		ev = &events{
			Factory: sess,
			log:     log,
		}
		go ev.gc(cfg)
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

func (e *events) gc(cfg Config) {
	if cfg.Retention == 0 {
		return
	}
	ticker := time.NewTicker(cfg.PollingPeriod)
	for {
		select {
		case <-ticker.C:
			sess := e.NewWriteSession()
			if err := sess.DeleteEvents(time.Now().Add(-cfg.Retention)); err != nil {
				e.log.Errorf("failed to delete old events: %v", err)
			}
		}
	}
}
