package deprovisioning

import (
	"errors"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/deprovisioning/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	smautomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestClsDeprovisionStepNoopRun(t *testing.T) {
	config := &cls.Config{}

	db := storage.NewMemoryStorage()
	repo := db.Operations()
	deprovisionerMock := &automock.ClsDeprovisioner{}

	step := NewClsDeprovisionStep(config, repo, deprovisionerMock)

	operation := internal.DeprovisioningOperation{
		Operation: internal.Operation{
			ID: "fake-skr-instance-id",
			ProvisioningParameters: internal.ProvisioningParameters{
				ErsContext: internal.ERSContext{GlobalAccountID: "fake-global-account-id"}},
			InstanceDetails: internal.InstanceDetails{
				Cls: internal.ClsData{
					Instance: internal.ServiceManagerInstanceInfo{
						Provisioned: false,
					}},
			},
		},
		SMClientFactory: servicemanager.NewFakeServiceManagerClientFactory(nil, nil),
	}

	_, offset, err := step.Run(operation, logger.NewLogDummy())
	require.Zero(t, offset)
	require.NoError(t, err)
}

func TestClsDeprovisionStepRun(t *testing.T) {
	var (
		globalAccountID = "fake-global-account-id"
		skrInstanceID   = "fake-skr-instance-id"
		clsInstance     = internal.ServiceManagerInstanceInfo{
			BrokerID:    "fake-broker-id",
			ServiceID:   "fake-service-id",
			PlanID:      "fake-plan-id",
			InstanceID:  "fake-instance-id",
			Provisioned: true,
		}
	)

	config := &cls.Config{
		ServiceManager: &cls.ServiceManagerConfig{
			Credentials: []*cls.ServiceManagerCredentials{
				{
					Region:   "eu",
					URL:      "https://foo.bar",
					Username: "fooUser",
					Password: "barPassword",
				},
			},
		},
	}

	db := storage.NewMemoryStorage()
	repo := db.Operations()

	smClient := &smautomock.Client{}
	operation := internal.DeprovisioningOperation{
		Operation: internal.Operation{
			InstanceID: skrInstanceID,
			ProvisioningParameters: internal.ProvisioningParameters{
				ErsContext: internal.ERSContext{GlobalAccountID: globalAccountID}},
			InstanceDetails: internal.InstanceDetails{
				Cls: internal.ClsData{Region: "eu", Instance: clsInstance},
			},
		},
		SMClientFactory: servicemanager.NewPassthroughServiceManagerClientFactory(smClient),
	}

	t.Run("triggering of deprovisioning fails", func(t *testing.T) {
		deprovisionerMock := &automock.ClsDeprovisioner{}
		deprovisionerMock.On("Deprovision", mock.Anything, &cls.DeprovisionRequest{
			GlobalAccountID: globalAccountID,
			SKRInstanceID:   skrInstanceID,
			Instance:        clsInstance.InstanceKey(),
		}).Return(errors.New("failure"))

		step := NewClsDeprovisionStep(config, repo, deprovisionerMock)

		op, offset, err := step.Run(operation, logger.NewLogDummy())
		require.True(t, op.Cls.Instance.Provisioned)
		require.NotEmpty(t, op.Cls.Instance.InstanceID)
		require.NotZero(t, offset)
		require.NoError(t, err)
	})

	t.Run("triggering of deprovisioning succeeds", func(t *testing.T) {
		deprovisionerMock := &automock.ClsDeprovisioner{}
		deprovisionerMock.On("Deprovision", mock.Anything, &cls.DeprovisionRequest{
			GlobalAccountID: globalAccountID,
			SKRInstanceID:   skrInstanceID,
			Instance:        clsInstance.InstanceKey(),
		}).Return(nil)

		step := NewClsDeprovisionStep(config, repo, deprovisionerMock)

		op, offset, err := step.Run(operation, logger.NewLogDummy())
		require.True(t, op.Cls.Instance.Provisioned)
		require.NotEmpty(t, op.Cls.Instance.InstanceID)
		require.NotZero(t, offset)
		require.NoError(t, err)
	})

	t.Run("deprovisioning fails", func(t *testing.T) {
		operation.Cls.Instance.DeprovisioningTriggered = true

		smClient.On("LastInstanceOperation", operation.Cls.Instance.InstanceKey(), "").Return(servicemanager.LastOperationResponse{
			State: servicemanager.Failed,
		}, nil)

		step := NewClsDeprovisionStep(config, repo, &automock.ClsDeprovisioner{})

		op, offset, err := step.Run(operation, logger.NewLogDummy())
		require.True(t, op.Cls.Instance.Provisioned)
		require.NotEmpty(t, op.Cls.Instance.InstanceID)
		require.NotZero(t, offset)
		require.NoError(t, err)
	})

	t.Run("deprovisioning in progress", func(t *testing.T) {
		operation.Cls.Instance.DeprovisioningTriggered = true

		smClient.On("LastInstanceOperation", operation.Cls.Instance.InstanceKey(), "").Return(servicemanager.LastOperationResponse{
			State: servicemanager.InProgress,
		}, nil)

		step := NewClsDeprovisionStep(config, repo, &automock.ClsDeprovisioner{})

		op, offset, err := step.Run(operation, logger.NewLogDummy())
		require.True(t, op.Cls.Instance.Provisioned)
		require.NotEmpty(t, op.Cls.Instance.InstanceID)
		require.NotZero(t, offset)
		require.NoError(t, err)
	})

	t.Run("deprovisioning succeeds", func(t *testing.T) {
		operation.Cls.Instance.DeprovisioningTriggered = true

		smClient.On("LastInstanceOperation", operation.Cls.Instance.InstanceKey(), "").Return(servicemanager.LastOperationResponse{
			State: servicemanager.InProgress,
		}, nil)

		step := NewClsDeprovisionStep(config, repo, &automock.ClsDeprovisioner{})

		op, offset, err := step.Run(operation, logger.NewLogDummy())
		require.True(t, op.Cls.Instance.Provisioned)
		require.NotEmpty(t, op.Cls.Instance.InstanceID)
		require.NotZero(t, offset)
		require.NoError(t, err)
	})

}
