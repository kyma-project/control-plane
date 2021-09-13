package connectivity_bind

import (
	"encoding/json"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"

	"github.com/stretchr/testify/assert"
)

const connectivityServiceKey = `
	{
		"clientid" : "clientid314159265359",
		"clientsecret": "Y2xpZW50c2VjcmV0Cg==",
		"connectivity_service":
			{
				"CAs_path": "/api/v1/CAs",
				"CAs_signing_path": "/api/v1/CAs/signing",
				"api_path": "/api/v1/CAs/signing",
				"tunnel_path": "/api/v1/tunnel",
				"url": "https://connectivity.company.com"
			},
		"subaccount_id": "db4dc3bd-3cb7-42d4-a6c0-23aa7842cb7d",
		"subaccount_subdomain": "some-subaccount-subdomain",
		"token_service_domain": "authentication.company.com",
		"token_service_url": "https://authentication.company.com/oauth/token",
		"token_service_url_pattern": "https://{tenant}.authentication.company.com/oauth/token",
		"token_service_url_pattern_tenant_key": "subaccount_subdomain",
		"xsappname": "xsappname314159265359"
	}
`

func TestConnectivityEncryptDecrypt(t *testing.T) {
	// given
	secretKey := "1234567890123456"
	givenOverrides := ConnectivityConfig{
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

	// when
	encryptedOverrides, err := EncryptConnectivityConfig(secretKey, &givenOverrides)
	assert.NoError(t, err)
	decryptedOverrides, err := DecryptConnectivityConfig(secretKey, encryptedOverrides)
	assert.NoError(t, err)

	// then
	assert.Equal(t, givenOverrides, *decryptedOverrides)
}

func TestConnectivityGetCredentials(t *testing.T) {
	// given
	binding := servicemanager.Binding{}

	// when
	err := json.Unmarshal([]byte(connectivityServiceKey), &binding.Credentials)
	assert.NoError(t, err)
	assert.NotNil(t, binding.Credentials)

	// then
	connOverrides, err := GetConnectivityCredentials(binding)
	assert.NoError(t, err)
	assert.NotNil(t, connOverrides)
	assert.Equal(t, "clientid314159265359", connOverrides.ClientId)
	assert.Equal(t, "Y2xpZW50c2VjcmV0Cg==", connOverrides.ClientSecret)
	assert.Equal(t, "/api/v1/CAs", connOverrides.ConnectivityService.CAsPath)
	assert.Equal(t, "/api/v1/CAs/signing", connOverrides.ConnectivityService.CAsSigningPath)
	assert.Equal(t, "/api/v1/CAs/signing", connOverrides.ConnectivityService.ApiPath)
	assert.Equal(t, "/api/v1/tunnel", connOverrides.ConnectivityService.TunnelPath)
	assert.Equal(t, "https://connectivity.company.com", connOverrides.ConnectivityService.Url)
	assert.Equal(t, "db4dc3bd-3cb7-42d4-a6c0-23aa7842cb7d", connOverrides.SubaccountId)
	assert.Equal(t, "some-subaccount-subdomain", connOverrides.SubaccountSubdomain)
	assert.Equal(t, "authentication.company.com", connOverrides.TokenServiceDomain)
	assert.Equal(t, "https://authentication.company.com/oauth/token", connOverrides.TokenServiceUrl)
	assert.Equal(t, "https://{tenant}.authentication.company.com/oauth/token", connOverrides.TokenServiceUrlPattern)
	assert.Equal(t, "subaccount_subdomain", connOverrides.TokenServiceUrlPatternTenantKey)
	assert.Equal(t, "xsappname314159265359", connOverrides.Xsappname)
}
