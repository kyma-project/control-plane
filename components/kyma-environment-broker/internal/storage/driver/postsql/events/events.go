package events

import (
	"fmt"
	"time"

	eventsapi "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/events"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
	"github.com/sirupsen/logrus"
)

type events struct {
	postsql.Factory

	log logrus.FieldLogger
}

func New(fac postsql.Factory, log logrus.FieldLogger) *events {
	return &events{Factory: fac, log: log}
}

func (e *events) ListEvents(filter eventsapi.EventFilter) ([]eventsapi.EventDTO, error) {
	if e == nil {
		return nil, fmt.Errorf("events are disabled")
	}
	sess := e.NewReadSession()
	return sess.ListEvents(filter)
}

func (e *events) InsertEvent(eventLevel eventsapi.EventLevel, message, instanceID, operationID string) {
	if e == nil {
		return
	}
	sess := e.NewWriteSession()
	if err := sess.InsertEvent(eventLevel, message, instanceID, operationID); err != nil {
		e.log.Errorf("failed to insert event [%v] %v/%v %q: %v", eventLevel, instanceID, operationID, message, err)
	}
}

func (e *events) RunGarbageCollection(pollingPeriod, retention time.Duration) {
	if e == nil {
		return
	}
	if retention == 0 {
		return
	}
	ticker := time.NewTicker(pollingPeriod)
	for {
		select {
		case <-ticker.C:
			sess := e.NewWriteSession()
			if err := sess.DeleteEvents(time.Now().Add(-retention)); err != nil {
				e.log.Errorf("failed to delete old events: %v", err)
			}
		}
	}
}
