package provisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession/mocks"

	"github.com/kyma-incubator/hydroform/install/installation"
	installationMocks "github.com/kyma-project/control-plane/components/provisioner/internal/installation/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWaitForInstallationStep_Run(t *testing.T) {

	cluster := model.Cluster{
		Kubeconfig: util.StringPtr(kubeconfig),
	}

	operation := model.Operation{
		ID:    "id",
		State: model.InProgress,
	}

	for _, testCase := range []struct {
		description          string
		installationMockFunc func(installationSvc *installationMocks.Service)
		expectedStage        model.OperationStage
		expectedDelay        time.Duration
	}{
		{
			description: "should continue installation if Installation error occurred",
			installationMockFunc: func(installationSvc *installationMocks.Service) {
				installationSvc.On("CheckInstallationState", mock.AnythingOfType("*rest.Config")).
					Return(installation.InstallationState{}, installation.InstallationError{ShortMessage: "error"})
			},
			expectedStage: model.WaitingForInstallation,
			expectedDelay: 30 * time.Second,
		},
		{
			description: "should continue installation if still in progress",
			installationMockFunc: func(installationSvc *installationMocks.Service) {
				installationSvc.On("CheckInstallationState", mock.AnythingOfType("*rest.Config")).
					Return(installation.InstallationState{State: "InProgress"}, nil)
			},
			expectedStage: model.WaitingForInstallation,
			expectedDelay: 30 * time.Second,
		},
		{
			description: "should go to the next stage if Kyma installed",
			installationMockFunc: func(installationSvc *installationMocks.Service) {
				installationSvc.On("CheckInstallationState", mock.AnythingOfType("*rest.Config")).
					Return(installation.InstallationState{State: "Installed"}, nil)
			},
			expectedStage: nextStageName,
			expectedDelay: 0,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			installationSvc := &installationMocks.Service{}
			session := &mocks.WriteSession{}

			session.On("UpdateOperationState", operation.ID, mock.AnythingOfType("string"),
				operation.State, mock.AnythingOfType("time.Time")).Return(nil).Once()

			testCase.installationMockFunc(installationSvc)

			waitForInstallationStep := NewWaitForInstallationStep(installationSvc, nextStageName, 10*time.Minute, session)

			// when
			result, err := waitForInstallationStep.Run(cluster, operation, logrus.New())

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedStage, result.Stage)
			assert.Equal(t, testCase.expectedDelay, result.Delay)
			installationSvc.AssertExpectations(t)
			session.AssertExpectations(t)
		})
	}

	t.Run("should return error if installation not started", func(t *testing.T) {
		// given
		installationSvc := &installationMocks.Service{}
		installationSvc.On("CheckInstallationState", mock.AnythingOfType("*rest.Config")).
			Return(installation.InstallationState{State: installation.NoInstallationState}, nil)

		session := &mocks.WriteSession{}

		waitForInstallationStep := NewWaitForInstallationStep(installationSvc, nextStageName, 10*time.Minute, session)

		// when
		_, err := waitForInstallationStep.Run(cluster, model.Operation{}, logrus.New())

		// then
		require.Error(t, err)
		installationSvc.AssertExpectations(t)
	})

}
