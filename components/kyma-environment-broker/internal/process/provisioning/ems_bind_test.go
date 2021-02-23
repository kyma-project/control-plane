package provisioning

import (
	"encoding/json"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"

	"github.com/stretchr/testify/assert"
)

func TestEmsEncryptDecrypt(t *testing.T) {
	// given
	secretKey := "1234567890123456"
	overridesIn := EventingOverrides{
		OauthClientId:      "oauthclientid",
		OauthClientSecret:  "oauthclientsecret",
		OauthTokenEndpoint: "oauthtokenendpoint",
		PublishUrl:         "publishurl",
		BebNamespace:       "bebnamespace",
	}

	// when
	encrypted, err := EncryptEventingOverrides(secretKey, &overridesIn)
	assert.NoError(t, err)
	overridesOut, err := DecryptEventingOverrides(secretKey, encrypted)
	assert.NoError(t, err)

	// then
	assert.Equal(t, overridesIn, *overridesOut)
}

func TestEmsGetCredentials(t *testing.T) {
	// given
	binding := servicemanager.Binding{}

	// when
	err := json.Unmarshal([]byte(serviceKey), &binding.Credentials)
	assert.NoError(t, err)
	assert.NotNil(t, binding.Credentials)

	// then
	eventingOverrides, err := GetEventingCredentials(binding)
	assert.NoError(t, err)
	assert.NotNil(t, eventingOverrides)
	assert.Equal(t, "messaging-httprest-oa2-clientid", eventingOverrides.OauthClientId)
	assert.Equal(t, "messaging-httprest-oa2-clientsecret", eventingOverrides.OauthClientSecret)
	assert.Equal(t, "https://messaging-httprest-oa2-tokenendpoint", eventingOverrides.OauthTokenEndpoint)
	assert.Equal(t, "https://messaging-httprest-oa2-uri", eventingOverrides.PublishUrl)
	assert.Equal(t, "kyma-namespace", eventingOverrides.BebNamespace)
}

const serviceKey = `
  {
  "management": [
    {
      "oa2": {
        "clientid": "management-oa2-clientid",
        "clientsecret": "management-oa2-clientsecret",
        "granttype": "management-oa2-granttype",
        "tokenendpoint": "management-oa2-tokenendpoint"
      },
      "uri": "https://management-uri"
    }
  ],
  "messaging": [
    {
      "broker": {
        "type": "sapmgw"
      },
      "oa2": {
        "clientid": "messaging-amqp10ws-oa2-clientid",
        "clientsecret": "messaging-amqp10ws-oa2-clientsecret",
        "granttype": "messaging-amqp10ws-oa2-granttype",
        "tokenendpoint": "https://messaging-amqp10ws-oa2-tokenendpoint"
      },
      "protocol": [
        "amqp10ws"
      ],
      "uri": "wss://messaging-amqp10ws-oa2-uri"
    },
    {
      "broker": {
        "type": "sapmgw"
      },
      "oa2": {
        "clientid": "messaging-mqtt311ws-oa2-clientid",
        "clientsecret": "messaging-mqtt311ws-oa2-clientsecret",
        "granttype": "messaging-mqtt311ws-oa2-granttype",
        "tokenendpoint": "https://messaging-mqtt311ws-oa2-tokenendpoint"
      },
      "protocol": [
        "mqtt311ws"
      ],
      "uri": "wss://messaging-mqtt311ws-oa2-uri"
    },
    {
      "broker": {
        "type": "saprestmgw"
      },
      "oa2": {
        "clientid": "messaging-httprest-oa2-clientid",
        "clientsecret": "messaging-httprest-oa2-clientsecret",
        "granttype": "messaging-httprest-oa2-granttype",
        "tokenendpoint": "https://messaging-httprest-oa2-tokenendpoint"
      },
      "protocol": [
        "httprest"
      ],
      "uri": "https://messaging-httprest-oa2-uri"
    }
  ],
  "namespace": "kyma-namespace",
  "serviceinstanceid": "serviceinstanceid",
  "xsappname": "xsappname"
}
`
