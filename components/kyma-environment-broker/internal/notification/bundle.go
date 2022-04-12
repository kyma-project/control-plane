package notification

import (
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
)

type (
	Config struct {
		Url      string `envconfig:"default="`
		Disabled bool   `envconfig:"default=true"`
	}

	NotificationClient interface {
		CreateEvent(CreateEventRequest) error
		UpdateEvent(UpdateEventRequest) error
		CancelEvent(CancelEventRequest) error
	}

	NotificationParams struct {
		OrchestrationID string               `json:"orchestrationId"`
		EventType       string               `json:"eventType"`
		Tenants         []NotificationTenant `json:"tenants"`
	}

	NotificationBundle struct {
		client             NotificationClient
		config             Config
		notificationParams NotificationParams
	}
)

func NewNotificationBundle(bundleIdentifier string, notificationParams NotificationParams, c NotificationClient, cfg Config) *NotificationBundle {
	return &NotificationBundle{
		client:             c,
		config:             cfg,
		notificationParams: notificationParams,
	}
}

func (b *NotificationBundle) CreateNotificationEvent() error {
	request := CreateEventRequest{
		OrchestrationID: b.notificationParams.OrchestrationID,
		EventType:       b.notificationParams.EventType,
		Tenants:         b.notificationParams.Tenants,
	}
	err := b.client.CreateEvent(request)
	if err != nil {
		return kebError.NewTemporaryError("failed to create event")
	}

	return nil
}

func (b *NotificationBundle) UpdateNotificationEvent() error {
	request := UpdateEventRequest{
		OrchestrationID: b.notificationParams.OrchestrationID,
		Tenants:         b.notificationParams.Tenants,
	}
	err := b.client.UpdateEvent(request)
	if err != nil {
		return kebError.NewTemporaryError("failed to update event")
	}

	return nil
}

func (b *NotificationBundle) CancelNotificationEvent() error {
	request := CancelEventRequest{OrchestrationID: b.notificationParams.OrchestrationID}
	err := b.client.CancelEvent(request)
	if err != nil {
		return kebError.NewTemporaryError("failed to cancel event")
	}

	return nil
}
