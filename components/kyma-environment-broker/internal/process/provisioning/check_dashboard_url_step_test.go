package provisioning

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	error2 "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

func TestCheckDashboardURLStep_Run(t *testing.T) {
	// given
	directorClient := &automock.DirectorClient{}
	directorClient.On("GetConsoleURL", statusGlobalAccountID, statusRuntimeID).Return(dashboardURL, nil)

	st := storage.NewMemoryStorage()
	operation := fixOperationRuntimeStatus(broker.GCPPlanID, internal.GCP)
	operation.RuntimeID = statusRuntimeID
	operation.DashboardURL = dashboardURL
	err := st.Operations().InsertProvisioningOperation(operation)
	assert.NoError(t, err)

	step := NewCheckDashboardURLStep(st.Operations(), directorClient, time.Second)

	// when
	operation, repeat, err := step.Run(operation, logrus.New())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
}

func TestCheckDashboardURLStep_RunRetry(t *testing.T) {
	// given
	directorClient := &automock.DirectorClient{}
	directorClient.On("GetConsoleURL", statusGlobalAccountID, statusRuntimeID).Return("", error2.NewTemporaryError("temporary error"))

	st := storage.NewMemoryStorage()
	operation := fixOperationRuntimeStatus(broker.GCPPlanID, internal.GCP)
	operation.RuntimeID = statusRuntimeID
	operation.DashboardURL = dashboardURL
	err := st.Operations().InsertProvisioningOperation(operation)
	assert.NoError(t, err)

	step := NewCheckDashboardURLStep(st.Operations(), directorClient, time.Second)

	// when
	operation, repeat, err := step.Run(operation, logrus.New())

	// then
	assert.NoError(t, err)
	assert.NotZero(t, repeat)
}
