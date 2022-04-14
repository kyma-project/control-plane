package upgrade_kyma

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/notification"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/notification/mocks"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/stretchr/testify/assert"
)

func TestSendNotificationStep_Run(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	tenants := []notification.NotificationTenant{
		{
			InstanceID: notification.FakeInstanceID,
			StartDate:  time.Now().Format("2006-01-02 15:04:05"),
			State:      notification.UnderMaintenanceEventState,
		},
	}
	paras := notification.NotificationParams{
		OrchestrationID: notification.FakeOrchestrationID,
		Tenants:         tenants,
	}

	bundleBuilder := &mocks.BundleBuilder{}
	bundle := &mocks.Bundle{}
	bundleBuilder.On("NewBundle", notification.FakeOrchestrationID, paras).Return(bundle, nil).Once()
	bundle.On("UpdateNotificationEvent").Return(nil).Once()

	operation := internal.UpgradeKymaOperation{
		Operation: internal.Operation{
			InstanceID:      notification.FakeInstanceID,
			OrchestrationID: notification.FakeOrchestrationID,
		},
	}
	step := NewSendNotificationStep(memoryStorage.Operations(), bundleBuilder)

	// when
	_, repeat, err := step.Run(operation, logger.NewLogDummy())

	// then
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
}
