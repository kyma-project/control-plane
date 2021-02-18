package cls

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestClsEncryptDecrypt(t *testing.T) {
	// given
	secretKey := "1234567890123456"
	overridesIn := ClsOverrides{
		FluentdEndPoint: "foo.bar",
		FluentdPassword: "fooPass",
		FluentdUsername: "fooUser",
		KibanaUrl:       "Kiib.url",
	}

	// when
	encrypted, err := EncryptOverrides(secretKey, &overridesIn)
	assert.NoError(t, err)
	overridesOut, err := DecryptOverrides(secretKey, encrypted)
	assert.NoError(t, err)

	// then
	assert.Equal(t, overridesIn, *overridesOut)
}
