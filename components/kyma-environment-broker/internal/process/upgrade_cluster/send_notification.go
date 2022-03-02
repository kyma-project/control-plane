package upgrade_cluster

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/notification"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

type SendNotificationStep struct {
	operationManager *process.UpgradeClusterOperationManager
	bundleBuilder    notification.BundleBuilder
}

func (s *SendNotificationStep) Name() string {
	return "Send_Notification"
}

func NewSendNotificationStep(os storage.Operations, bundleBuilder notification.BundleBuilder) *SendNotificationStep {
	return &SendNotificationStep{
		operationManager: process.NewUpgradeClusterOperationManager(os),
		bundleBuilder:    bundleBuilder,
	}
}

func (s *SendNotificationStep) Run(operation internal.UpgradeClusterOperation, log logrus.FieldLogger) (internal.UpgradeClusterOperation, time.Duration, error) {
	tenants := []notification.NotificationTenant{
		{
			InstanceID: operation.InstanceID,
			StartDate:  time.Now().Format("2006-01-02 15:04:05"),
			State:      notification.UnderMaintenanceEventState,
		},
	}
	notificationParams := notification.NotificationParams{
		OrchestrationID: operation.OrchestrationID,
		Tenants:         tenants,
	}
	notificationBundle, err := s.bundleBuilder.NewBundle(operation.OrchestrationID, notificationParams)
	if err != nil {
		log.Errorf("%s: %s", "Failed to create Notification Bundle", err)
		return operation, 5 * time.Second, nil
	}

	log.Infof("Sending http request to customer notification service")
	err = notificationBundle.UpdateNotificationEvent()
	//currently notification error can only be temporary error
	if err != nil && kebError.IsTemporaryError(err) {
		msg := fmt.Sprintf("cannot update notification for orchestration %s", operation.OrchestrationID)
		log.Errorf("%s: %s", msg, err)
		return operation, 5 * time.Second, nil
	}

	return operation, 0, nil
}
