package notification

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceProviderBundle_CreateNotificationEvent(t *testing.T) {
	// given
	tenants := []NotificationTenant{
		{
			InstanceID: "WEAJKG-INSTANCE-1",
			StartDate:  "2022-01-01T20:00:02Z",
		},
	}
	paras := NotificationParams{
		OrchestrationID: "test-create",
		EventType:       KymaMaintenanceNumber,
		Tenants:         tenants,
	}

	client := NewFakeClient()
	bundle := NewNotificationBundle("ochstA", paras, client, Config{Disabled: false})

	// when
	err := bundle.CreateNotificationEvent()

	// then
	assert.NoError(t, err)

	event, err := client.GetMaintenanceEvent("test-create")
	assert.NoError(t, err)
	assert.Equal(t, "test-create", event.OrchestrationID)
	assert.Equal(t, "WEAJKG-INSTANCE-1", event.Tenants[0].InstanceID)
	assert.Equal(t, "2022-01-01T20:00:02Z", event.Tenants[0].StartDate)
	assert.Equal(t, "1", event.EventType)
}

func TestServiceProviderBundle_UpdateNotificationEvent(t *testing.T) {
	// given
	tenants := []NotificationTenant{
		{
			InstanceID: "WEAJKG-INSTANCE-1",
			StartDate:  "2022-01-01T20:00:02Z",
			State:      UnderMaintenanceEventState,
		},
	}
	paras := NotificationParams{
		OrchestrationID: FakeOrchestrationID,
		Tenants:         tenants,
	}

	client := NewFakeClient()
	bundle := NewNotificationBundle("ochstA", paras, client, Config{Disabled: false})

	// when
	err := bundle.UpdateNotificationEvent()

	// then
	assert.NoError(t, err)

	event, err := client.GetMaintenanceEvent(FakeOrchestrationID)
	assert.NoError(t, err)
	assert.Equal(t, "1", event.Tenants[0].State)
	assert.Equal(t, "2022-01-01T20:00:02Z", event.Tenants[0].StartDate)
}

func TestServiceProviderBundle_CancelNotificationEvent(t *testing.T) {
	// given
	paras := NotificationParams{
		OrchestrationID: FakeOrchestrationID,
	}

	client := NewFakeClient()
	bundle := NewNotificationBundle("ochstA", paras, client, Config{Disabled: false})

	// when
	err := bundle.CancelNotificationEvent()

	// then
	assert.NoError(t, err)

	event, err := client.GetMaintenanceEvent(FakeOrchestrationID)
	assert.NoError(t, err)
	assert.Equal(t, "3", event.Tenants[0].State)
}
