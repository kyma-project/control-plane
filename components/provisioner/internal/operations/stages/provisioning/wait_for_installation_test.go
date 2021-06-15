package provisioning

import (
	"errors"
	"testing"
	"time"

	kymaInstallation "github.com/kyma-project/control-plane/components/provisioner/internal/installation"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
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
		ID:         clusterID,
		Kubeconfig: util.StringPtr(kubeconfig),
		KymaConfig: model.KymaConfig{Installer: model.KymaOperatorInstaller},
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
			description: "should continue installation if recoverable Installation error occurred",
			installationMockFunc: func(installationSvc *installationMocks.Service) {
				installationSvc.On("CheckInstallationState", clusterID, mock.AnythingOfType("*rest.Config")).
					Return(installation.InstallationState{}, installation.InstallationError{ShortMessage: "error", Recoverable: true})
			},
			expectedStage: model.WaitingForInstallation,
			expectedDelay: 30 * time.Second,
		},
		{
			description: "should continue installation if still in progress",
			installationMockFunc: func(installationSvc *installationMocks.Service) {
				installationSvc.On("CheckInstallationState", clusterID, mock.AnythingOfType("*rest.Config")).
					Return(installation.InstallationState{State: "InProgress"}, nil)
			},
			expectedStage: model.WaitingForInstallation,
			expectedDelay: 30 * time.Second,
		},
		{
			description: "should go to the next stage if Kyma installed",
			installationMockFunc: func(installationSvc *installationMocks.Service) {
				installationSvc.On("CheckInstallationState", clusterID, mock.AnythingOfType("*rest.Config")).
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
			installationSvcs := map[model.KymaInstaller]kymaInstallation.Service{
				model.KymaOperatorInstaller: installationSvc,
			}

			waitForInstallationStep := NewWaitForInstallationStep(installationSvcs, nextStageName, 10*time.Minute, session)

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
		installationSvc.On("CheckInstallationState", clusterID, mock.AnythingOfType("*rest.Config")).
			Return(installation.InstallationState{State: installation.NoInstallationState}, nil)

		session := &mocks.WriteSession{}
		installationSvcs := map[model.KymaInstaller]kymaInstallation.Service{
			model.KymaOperatorInstaller: installationSvc,
		}

		waitForInstallationStep := NewWaitForInstallationStep(installationSvcs, nextStageName, 10*time.Minute, session)

		// when
		_, err := waitForInstallationStep.Run(cluster, model.Operation{}, logrus.New())

		// then
		require.Error(t, err)
		installationSvc.AssertExpectations(t)
	})

	t.Run("should return error if parallel installation is in unrecoverable error state", func(t *testing.T) {
		// given
		cluster.KymaConfig.Installer = model.ParallelInstaller
		installationSvc := &installationMocks.Service{}
		installationSvc.On("CheckInstallationState", clusterID, mock.AnythingOfType("*rest.Config")).
			Return(installation.InstallationState{}, installation.InstallationError{ShortMessage: "error", Recoverable: false})

		session := &mocks.WriteSession{}
		session.On("UpdateOperationState", operation.ID, mock.AnythingOfType("string"),
			operation.State, mock.AnythingOfType("time.Time")).Return(nil).Once()
		installationSvcs := map[model.KymaInstaller]kymaInstallation.Service{
			model.ParallelInstaller: installationSvc,
		}

		waitForInstallationStep := NewWaitForInstallationStep(installationSvcs, nextStageName, 10*time.Minute, session)

		// when
		_, err := waitForInstallationStep.Run(cluster, operation, logrus.New())

		// then
		require.Error(t, err)
		assert.True(t, errors.As(err, &operations.NonRecoverableError{}))
		installationSvc.AssertExpectations(t)
		session.AssertExpectations(t)
	})
}
