package provisioning

import (
	"testing"

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
	encrypted, err := encryptOverrides(secretKey, &overridesIn)
	assert.NoError(t, err)
	overridesOut, err := decryptOverrides(secretKey, encrypted)
	assert.NoError(t, err)

	// then
	assert.Equal(t, overridesIn, *overridesOut)
}
