package provisioning

import (
	"testing"
	"time"

	automock2 "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ias"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ias/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"

	"github.com/stretchr/testify/assert"
)

const (
	iasTypeInstanceID   = "1180670b-9de4-421b-8f76-919faeb34249"
	iasTypeURLDashboard = "http://example.com"
)

func TestIASType_ConfigureType(t *testing.T) {
	// given
	bundleBuilder := &automock.BundleBuilder{}
	defer bundleBuilder.AssertExpectations(t)

	for inputID := range ias.ServiceProviderInputs {
		bundle := &automock.Bundle{}
		defer bundle.AssertExpectations(t)
		bundle.On("FetchServiceProviderData").Return(nil).Once()
		bundle.On("ServiceProviderName").Return("MockProviderName")
		bundle.On("ConfigureServiceProviderType", iasTypeURLDashboard).Return(nil).Once()
		bundleBuilder.On("NewBundle", iasTypeInstanceID, inputID).Return(bundle, nil).Once()
	}
	directorClient := &automock2.DirectorClient{}
	directorClient.On("SetLabel", statusGlobalAccountID, statusRuntimeID, grafanaURLLabel, "https://grafana.kyma.org").Return(nil)
	defer directorClient.AssertExpectations(t)

	step := NewIASTypeStep(bundleBuilder, directorClient)

	// when
	_, repeat, err := step.Run(internal.ProvisioningOperation{
		Operation: internal.Operation{
			InstanceDetails: internal.InstanceDetails{
				ShootDomain: "kyma.org",
				RuntimeID:   statusRuntimeID,
			},
			ProvisioningParameters: internal.ProvisioningParameters{
				ErsContext: internal.ERSContext{
					GlobalAccountID: statusGlobalAccountID,
				},
			},
			InstanceID: iasTypeInstanceID,
		},
		DashboardURL: iasTypeURLDashboard,
	}, logger.NewLogDummy())

	// then
	assert.Equal(t, time.Duration(0), repeat)
	assert.NoError(t, err)
}
