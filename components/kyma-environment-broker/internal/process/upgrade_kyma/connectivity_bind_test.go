package upgrade_kyma

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/connectivity_bind"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUpgradeConnectivityBind_Run(t *testing.T) {
	t.Run("Should succesfully apply overrides", func(t *testing.T) {
		connectivityOverrides := &connectivity_bind.ConnectivityConfig{

			ClientId:     "clientid",
			ClientSecret: "clientsecret",
			ConnectivityService: struct {
				CAsPath        string `json:"CAs_path"`
				CAsSigningPath string `json:"CAs_signing_path"`
				ApiPath        string `json:"api_path"`
				TunnelPath     string `json:"tunnel_path"`
				Url            string `json:"url"`
			}{
				CAsPath:        "caspath",
				CAsSigningPath: "passigningpath",
				ApiPath:        "apipath",
				TunnelPath:     "tunnelpath",
				Url:            "url",
			},
			SubaccountId:                    "subaccoutid",
			SubaccountSubdomain:             "subaccountsubdomain",
			TokenServiceDomain:              "tokenservicedomain",
			TokenServiceUrl:                 "tokenserviceurl",
			TokenServiceUrlPattern:          "tokenserviceurlpattern",
			TokenServiceUrlPatternTenantKey: "tokenserviceurlpatterntenantkey",
			Xsappname:                       "xsappname",
		}

		expectedConfigEntryInput := []*gqlschema.ConfigEntryInput{
			{
				Key:   "connectivityProxyServiceKey.clientid",
				Value: connectivityOverrides.ClientId,
			},
			{
				Key:   "connectivityProxyServiceKey.clientsecret",
				Value: connectivityOverrides.ClientSecret,
			},
			{
				Key:   "connectivityProxyServiceKey.connectivityServiceUrl",
				Value: connectivityOverrides.ConnectivityService.Url,
			},
			{
				Key:   "connectivityProxyServiceKey.subaccountId",
				Value: connectivityOverrides.SubaccountId,
			},
			{
				Key:   "connectivityProxyServiceKey.subaccountSubdomain",
				Value: connectivityOverrides.SubaccountSubdomain,
			},
			{
				Key:   "connectivityProxyServiceKey.tokenServiceDomain",
				Value: connectivityOverrides.TokenServiceDomain,
			},
			{
				Key:   "connectivityProxyServiceKey.tokenServiceUrl",
				Value: connectivityOverrides.TokenServiceUrl,
			},
			{
				Key:   "connectivityProxyServiceKey.tokenServiceUrlPattern",
				Value: connectivityOverrides.TokenServiceUrlPattern,
			},
			{
				Key:   "connectivityProxyServiceKey.tokenServiceUrlPatternTenantKey",
				Value: connectivityOverrides.TokenServiceUrlPatternTenantKey,
			},
			{
				Key:   "connectivityProxyServiceKey.xsappname",
				Value: connectivityOverrides.Xsappname,
			},
		}

		secretKey := "1111111111111111"

		inputCreator := &automock.ProvisionerInputCreator{}
		inputCreator.On("AppendOverrides", connectivity_bind.ConnectivityProxyComponentName, expectedConfigEntryInput).Return(inputCreator)

		smClientFactory := servicemanager.NewFakeServiceManagerClientFactory(nil, nil)

		encryptedOverrides, err := connectivity_bind.EncryptConnectivityConfig(secretKey, connectivityOverrides)
		require.NoError(t, err)

		operation := internal.UpgradeKymaOperation{
			SMClientFactory: smClientFactory,
			InputCreator:    inputCreator,
			Operation: internal.Operation{
				InstanceDetails: internal.InstanceDetails{
					Connectivity: internal.ConnectivityData{
						Instance: internal.ServiceManagerInstanceInfo{
							ProvisioningTriggered: true,
							Provisioned:           true,
						},
						BindingID: "",
						Overrides: encryptedOverrides,
					},
				},
			},
		}

		step := NewConnectivityUpgradeBindStep(nil, secretKey)

		//when
		_, _, err = step.Run(operation, logger.NewLogDummy())
		require.NoError(t, err)

		//then
		inputCreator.AssertExpectations(t)
	})
}
