package events

import (
	"fmt"
	"sync"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Enabled       bool          `envconfig:"default=false"`
	Retention     time.Duration `envconfig:"default=72h"`
	PollingPeriod time.Duration `envconfig:"default=1h"`
}

var (
	ev       *events
	initLock sync.Mutex
)

type events struct {
	postsql.Factory

	log logrus.FieldLogger
}

func New(cfg Config, sess postsql.Factory, log logrus.FieldLogger) *events {
	if !cfg.Enabled {
		return nil
	}
	initLock.Lock()
	defer initLock.Unlock()
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

func (e *events) ListEvents(filter dbmodel.EventFilter) ([]dbmodel.EventDTO, error) {
	if e != nil {
		sess := e.NewReadSession()
		return sess.ListEvents(filter)
	} else {
		return nil, fmt.Errorf("events are disabled")
	}
}

func (e *events) InsertEvent(eventLevel dbmodel.EventLevel, message, instanceID, operationID string) {
	if e != nil {
		sess := e.NewWriteSession()
		if err := sess.InsertEvent(eventLevel, message, instanceID, operationID); err != nil {
			e.log.Errorf("failed to insert event [%v] %v/%v %q: %v", eventLevel, instanceID, operationID, message, err)
		}
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
