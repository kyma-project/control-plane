package dbmodel

import (
	"time"
)

type EventLevel string

const (
	InfoEventLevel  EventLevel = "info"
	ErrorEventLevel EventLevel = "error"
)

type EventDTO struct {
	ID          string
	Level       EventLevel
	InstanceID  *string
	OperationID *string
	Message     string
	CreatedAt   time.Time
}

type EventFilter struct {
	InstanceIDs  []string
	OperationIDs []string
}
