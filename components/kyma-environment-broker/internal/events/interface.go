package events

import (
	"fmt"
	"sync"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/events"
)

type Config struct {
	Enabled       bool          `envconfig:"default=false"`
	Retention     time.Duration `envconfig:"default=72h"`
	PollingPeriod time.Duration `envconfig:"default=1h"`
}

var (
	ev       Interface
	initLock sync.Mutex
)

type Interface interface {
	ListEvents(filter events.EventFilter) ([]events.EventDTO, error)
	InsertEvent(eventLevel events.EventLevel, message, instanceID, operationID string)
	RunGarbageCollection(pollingPeriod, retention time.Duration)
}

func New(cfg Config, events Interface) Interface {
	if !cfg.Enabled {
		return nil
	}
	initLock.Lock()
	defer initLock.Unlock()
	if ev == nil {
		ev = events
		go ev.RunGarbageCollection(cfg.PollingPeriod, cfg.Retention)
	}
	return ev
}

func Infof(instanceID, operationID, format string, args ...any) {
	insertEvent(events.InfoEventLevel, fmt.Sprintf(format, args...), instanceID, operationID)
}

func Errorf(instanceID, operationID string, err error, format string, args ...any) {
	insertEvent(events.ErrorEventLevel, fmt.Sprintf("%v: %v", fmt.Sprintf(format, args...), err), instanceID, operationID)
}

func insertEvent(eventLevel events.EventLevel, msg, instanceID, operationID string) {
	if ev != nil {
		ev.InsertEvent(eventLevel, msg, instanceID, operationID)
	}
}
