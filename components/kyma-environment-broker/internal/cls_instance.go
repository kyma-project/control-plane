package internal

import (
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type CLSInstanceOption func(*CLSInstance)

func WithVersion(version int) CLSInstanceOption {
	return func(i *CLSInstance) {
		i.version = version
	}
}

func WithID(id string) CLSInstanceOption {
	return func(i *CLSInstance) {
		i.id = id
	}
}

func WithCreatedAt(createdAt time.Time) CLSInstanceOption {
	return func(i *CLSInstance) {
		i.createdAt = createdAt
	}
}

func WithReferences(references ...string) CLSInstanceOption {
	return func(i *CLSInstance) {
		i.references = references
	}
}

func WithBeingRemovedBy(beingRemovedBy string) CLSInstanceOption {
	return func(i *CLSInstance) {
		i.beingRemovedBy = beingRemovedBy
	}
}

func defaultCLSInstanceOptions() []CLSInstanceOption {
	return []CLSInstanceOption{
		WithVersion(0),
		WithID(uuid.New().String()),
		WithCreatedAt(time.Now()),
		WithBeingRemovedBy(""),
	}
}

type CLSInstance struct {
	version         int
	id              string
	globalAccountID string
	region          string
	createdAt       time.Time
	references      []string
	beingRemovedBy  string

	events []interface{}
}
type CLSInstanceReferencedEvent struct {
	SKRInstanceID string
}

type CLSInstanceUnreferencedEvent struct {
	SKRInstanceID string
}

func NewCLSInstance(globalAccountID, region string, opts ...CLSInstanceOption) *CLSInstance {
	result := &CLSInstance{
		globalAccountID: globalAccountID,
		region:          region,
	}

	opts = append(defaultCLSInstanceOptions(), opts...)

	for _, opt := range opts {
		opt(result)
	}

	return result
}

func (i *CLSInstance) Version() int {
	return i.version
}

func (i *CLSInstance) ID() string {
	return i.id
}

func (i *CLSInstance) GlobalAccountID() string {
	return i.globalAccountID
}

func (i *CLSInstance) Region() string {
	return i.region
}

func (i *CLSInstance) CreatedAt() time.Time {
	return i.createdAt
}

func (i *CLSInstance) References() []string {
	return i.references
}

func (i *CLSInstance) IsBeingRemoved() bool {
	return len(i.beingRemovedBy) > 0
}

func (i *CLSInstance) BeingRemovedBy() string {
	return i.beingRemovedBy
}

func (i *CLSInstance) Events() []interface{} {
	return i.events
}

func (i *CLSInstance) AddReference(skrInstanceID string) {
	i.references = append(i.references, skrInstanceID)
	i.events = append(i.events, CLSInstanceReferencedEvent{SKRInstanceID: skrInstanceID})
}

func (i *CLSInstance) RemoveReference(skrInstanceID string) error {
	found := false
	idx := 0
	for i, refID := range i.references {
		if refID == skrInstanceID {
			idx = i
			found = true
			break
		}
	}

	if !found {
		return errors.New("not found")
	}

	i.references = append(i.references[:idx], i.references[idx+1:]...)
	i.events = append(i.events, CLSInstanceUnreferencedEvent{SKRInstanceID: skrInstanceID})

	if len(i.references) == 0 {
		i.beingRemovedBy = skrInstanceID
	}

	return nil
}

func (i *CLSInstance) IsReferencedBy(skrInstanceID string) bool {
	for _, refID := range i.references {
		if refID == skrInstanceID {
			return true
		}
	}
	return false
}
