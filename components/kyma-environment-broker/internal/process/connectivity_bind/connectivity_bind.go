package connectivity_bind

import (
	"encoding/json"
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
)

type ConnectivityConfig struct {
	ClientId            string `json:"clientid"`
	ClientSecret        string `json:"clientsecret"`
	ConnectivityService struct {
		CAsPath        string `json:"CAs_path"`
		CAsSigningPath string `json:"CAs_signing_path"`
		ApiPath        string `json:"api_path"`
		TunnelPath     string `json:"tunnel_path"`
		Url            string `json:"url"`
	} `json:"connectivity_service"`
	SubaccountId                    string `json:"subaccount_id"`
	SubaccountSubdomain             string `json:"subaccount_subdomain"`
	TokenServiceDomain              string `json:"token_service_domain"`
	TokenServiceUrl                 string `json:"token_service_url"`
	TokenServiceUrlPattern          string `json:"token_service_url_pattern"`
	TokenServiceUrlPatternTenantKey string `json:"token_service_url_pattern_tenant_key"`
	Xsappname                       string `json:"xsappname"`
}

const ConnectivityProxyComponentName = "connectivity-k8s-helm-xmake"

func PrepareOverrides(connectivityOverrides *ConnectivityConfig) []*gqlschema.ConfigEntryInput {
	overrides := []*gqlschema.ConfigEntryInput{
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
	return overrides
}

func GetConnectivityCredentials(binding servicemanager.Binding) (*ConnectivityConfig, error) {
	credentials := binding.Credentials
	csMap, ok := credentials["connectivity_service"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf(
			"failed to convert connectivity_service part of the credentials to map[string]interface{}")
	}

	return &ConnectivityConfig{
		ClientId:     credentials["clientid"].(string),
		ClientSecret: credentials["clientsecret"].(string),
		ConnectivityService: struct {
			CAsPath        string `json:"CAs_path"`
			CAsSigningPath string `json:"CAs_signing_path"`
			ApiPath        string `json:"api_path"`
			TunnelPath     string `json:"tunnel_path"`
			Url            string `json:"url"`
		}{
			CAsPath:        csMap["CAs_path"].(string),
			CAsSigningPath: csMap["CAs_signing_path"].(string),
			ApiPath:        csMap["CAs_signing_path"].(string),
			TunnelPath:     csMap["tunnel_path"].(string),
			Url:            csMap["url"].(string),
		},
		SubaccountId:                    credentials["subaccount_id"].(string),
		SubaccountSubdomain:             credentials["subaccount_subdomain"].(string),
		TokenServiceDomain:              credentials["token_service_domain"].(string),
		TokenServiceUrl:                 credentials["token_service_url"].(string),
		TokenServiceUrlPattern:          credentials["token_service_url_pattern"].(string),
		TokenServiceUrlPatternTenantKey: credentials["token_service_url_pattern_tenant_key"].(string),
		Xsappname:                       credentials["xsappname"].(string),
	}, nil
}

func EncryptConnectivityConfig(secretKey string, overrides *ConnectivityConfig) (string, error) {
	marshalledOverrides, err := json.Marshal(*overrides)
	if err != nil {
		return "", errors.Wrap(err, "while encoding connectivity overrides")
	}
	encryptedOverrides, err := storage.NewEncrypter(secretKey).Encrypt(marshalledOverrides)
	if err != nil {
		return "", errors.Wrap(err, "while encrypting connectivity overrides")
	}
	return string(encryptedOverrides), nil
}

func DecryptConnectivityConfig(secretKey string, encryptedOverrides string) (*ConnectivityConfig, error) {
	decryptedOverrides, err := storage.NewEncrypter(secretKey).Decrypt([]byte(encryptedOverrides))
	if err != nil {
		return nil, errors.Wrap(err, "while decrypting connectivity overrides")
	}
	connectivityOverrides := ConnectivityConfig{}
	if err := json.Unmarshal(decryptedOverrides, &connectivityOverrides); err != nil {
		return nil, errors.Wrap(err, "while unmarshalling connectivity overrides")
	}
	return &connectivityOverrides, nil
}
