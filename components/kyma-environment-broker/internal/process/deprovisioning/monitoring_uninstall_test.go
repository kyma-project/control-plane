package deprovisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/monitoring"
	monitoringmocks "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/monitoring/mocks"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestMonitoringUninstallStep_Run(t *testing.T) {
	t.Run("testing Monitoring Uninstall", func(t *testing.T) {
		// given
		repo := storage.NewMemoryStorage().Operations()
		cfg := monitoring.Config{
			Namespace:       "mornitoring",
			ChartUrl:        "notEmptyChart",
			RemoteWriteUrl:  "notEmptyUrl",
			RemoteWritePath: "notEmptyPath",
			Disabled:        false,
		}
		monitoringClient := &monitoringmocks.Client{}
		monitoringClient.On("IsPresent", "c-012345").Return(true, nil)
		monitoringClient.On("UninstallRelease", "c-012345").Return(nil, nil)
		operation := internal.DeprovisioningOperation{
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
		step := NewMonitoringUnistallStep(repo, monitoringClient, cfg)

		// when
		_, repeat, err := step.Run(operation, logger.NewLogDummy())

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		monitoringClient.AssertExpectations(t)
	})
}

func TestMonitoringUninstallSkipStep_Run(t *testing.T) {
	t.Run("testing Monitoring Uninstall Skip", func(t *testing.T) {
		// given
		repo := storage.NewMemoryStorage().Operations()
		cfg := monitoring.Config{
			Namespace:       "mornitoring",
			ChartUrl:        "notEmptyChart",
			RemoteWriteUrl:  "notEmptyUrl",
			RemoteWritePath: "notEmptyPath",
			Disabled:        false,
		}
		monitoringClient := &monitoringmocks.Client{}
		monitoringClient.On("IsPresent", "c-012345").Return(false, nil)
		operation := internal.DeprovisioningOperation{
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
		step := NewMonitoringUnistallStep(repo, monitoringClient, cfg)

		// when
		_, repeat, err := step.Run(operation, logger.NewLogDummy())

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		monitoringClient.AssertExpectations(t)
	})
}

func StringPtr(str string) *string {
	return &str
}
