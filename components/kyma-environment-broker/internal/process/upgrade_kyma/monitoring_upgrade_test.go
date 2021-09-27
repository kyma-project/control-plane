package upgrade_kyma

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/monitoring"
	monitoringmocks "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/monitoring/mocks"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/release"
)

func TestMonitoringUpgradeStep_Run(t *testing.T) {
	t.Run("testing Monitoring Upgrade", func(t *testing.T) {
		// given
		repo := storage.NewMemoryStorage().Operations()
		cfg := monitoring.Config{
			Namespace:       "mornitoring",
			ChartUrl:        "notEmptyChart",
			RemoteWriteUrl:  "notEmptyUrl",
			RemoteWritePath: "notEmptyPath",
			Disabled:        false,
		}
		upgradeResponse := &release.Release{
			Info: &release.Info{
				Status:      release.StatusDeployed,
				Description: "Installed",
			},
		}
		monitoringClient := &monitoringmocks.Client{}
		monitoringClient.On("IsPresent", "c-012345").Return(true, nil)
		monitoringClient.On("UpgradeRelease", mock.MatchedBy(func(params monitoring.Parameters) bool {
			return params.ReleaseName == "c-012345" &&
				params.InstanceID == "d3d5dca4-5dc8-44ee-a825-755c2a3fb839" &&
				params.GlobalAccountID == "d9d501c2-bdcb-49f2-8e86-1c4e05b90f5e" &&
				params.SubaccountID == "3cb65e5b-e455-4799-bf35-be46e8f5a533" &&
				params.ShootName == "c-012345" &&
				params.Region == "eastus" &&
				params.Username == "d3d5dca4-5dc8-44ee-a825-755c2a3fb839" &&
				params.Password == "0123456789"
		})).Return(upgradeResponse, nil)
		inputCreatorMock := &automock.ProvisionerInputCreator{}
		defer inputCreatorMock.AssertExpectations(t)
		inputCreatorMock.On("AppendOverrides", "rma", []*gqlschema.ConfigEntryInput{
			{
				Key:   "vmuser.username",
				Value: "d3d5dca4-5dc8-44ee-a825-755c2a3fb839",
			},
			{
				Key:   "vmuser.password",
				Value: "0123456789",
			},
		}).Return(nil).Once()
		operation := internal.UpgradeKymaOperation{
			InputCreator: inputCreatorMock,
			Operation: internal.Operation{
				InstanceID: "d3d5dca4-5dc8-44ee-a825-755c2a3fb839",
				InstanceDetails: internal.InstanceDetails{
					SubAccountID: "3cb65e5b-e455-4799-bf35-be46e8f5a533",
					ShootName:    "c-012345",
					Monitoring: internal.MonitoringData{
						Username: "d3d5dca4-5dc8-44ee-a825-755c2a3fb839",
						Password: "0123456789",
					},
				},
				ProvisioningParameters: internal.ProvisioningParameters{
					ErsContext: internal.ERSContext{GlobalAccountID: "d9d501c2-bdcb-49f2-8e86-1c4e05b90f5e"},
					Parameters: internal.ProvisioningParametersDTO{Region: StringPtr("eastus")},
				},
			},
		}
		step := NewMonitoringUpgradeStep(repo, monitoringClient, cfg)

		// when
		_, repeat, err := step.Run(operation, logger.NewLogDummy())

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		monitoringClient.AssertExpectations(t)
	})
}

func TestMonitoringUpgradeSkipStep_Run(t *testing.T) {
	t.Run("testing Monitoring Upgrade Install", func(t *testing.T) {
		// given
		repo := storage.NewMemoryStorage().Operations()
		cfg := monitoring.Config{
			Namespace:       "mornitoring",
			ChartUrl:        "notEmptyChart",
			RemoteWriteUrl:  "notEmptyUrl",
			RemoteWritePath: "notEmptyPath",
			Disabled:        false,
		}
		installationResponse := &release.Release{
			Info: &release.Info{
				Status:      release.StatusDeployed,
				Description: "Installed",
			},
		}
		inputCreatorMock := &automock.ProvisionerInputCreator{}
		defer inputCreatorMock.AssertExpectations(t)
		inputCreatorMock.On("AppendOverrides", "rma", mock.MatchedBy(func(overrides []*gqlschema.ConfigEntryInput) bool {
			return overrides[0].Key == "vmuser.username" &&
				overrides[0].Value == "d3d5dca4-5dc8-44ee-a825-755c2a3fb839" &&
				overrides[1].Key == "vmuser.password" &&
				overrides[1].Value != ""
		})).Return(nil).Once()
		monitoringClient := &monitoringmocks.Client{}
		monitoringClient.On("IsPresent", "c-012345").Return(false, nil)
		monitoringClient.On("InstallRelease", mock.MatchedBy(func(params monitoring.Parameters) bool {
			return params.ReleaseName == "c-012345" &&
				params.InstanceID == "d3d5dca4-5dc8-44ee-a825-755c2a3fb839" &&
				params.GlobalAccountID == "d9d501c2-bdcb-49f2-8e86-1c4e05b90f5e" &&
				params.SubaccountID == "3cb65e5b-e455-4799-bf35-be46e8f5a533" &&
				params.ShootName == "c-012345" &&
				params.Region == "eastus" &&
				params.Username == "d3d5dca4-5dc8-44ee-a825-755c2a3fb839" &&
				params.Password != ""
		})).Return(installationResponse, nil)

		operation := internal.UpgradeKymaOperation{
			InputCreator: inputCreatorMock,
			Operation: internal.Operation{
				InstanceID: "d3d5dca4-5dc8-44ee-a825-755c2a3fb839",
				InstanceDetails: internal.InstanceDetails{
					SubAccountID: "3cb65e5b-e455-4799-bf35-be46e8f5a533",
					ShootName:    "c-012345",
				},
				ProvisioningParameters: internal.ProvisioningParameters{
					ErsContext: internal.ERSContext{GlobalAccountID: "d9d501c2-bdcb-49f2-8e86-1c4e05b90f5e"},
					Parameters: internal.ProvisioningParametersDTO{Region: StringPtr("eastus")},
				},
			},
		}
		step := NewMonitoringUpgradeStep(repo, monitoringClient, cfg)
		err := repo.InsertUpgradeKymaOperation(operation)
		require.NoError(t, err)

		// when
		op, repeat, err := step.Run(operation, logger.NewLogDummy())

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		monitoringClient.AssertExpectations(t)
		assert.Equal(t, "d3d5dca4-5dc8-44ee-a825-755c2a3fb839", op.Monitoring.Username)
	})
}
