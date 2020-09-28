package storage

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewEncrypter(t *testing.T) {

	secretKey := "sdas@mlkasmfL_("

	e := NewEncrypter(secretKey)

	dto := struct {
		Data string `json:"data"`
	}{
		Data: secretKey,
	}

	j, err := json.Marshal(&dto)
	require.NoError(t, err)

	enc, err := e.Encrypt(string(j))
	require.NoError(t, err)
	assert.NotEqual(t, string(j), enc)

	enc, err = e.Decrypt(enc)
	require.NoError(t, err)
	assert.Equal(t, string(j), enc)

	err = json.Unmarshal([]byte(enc), &dto)
	require.NoError(t, err)
}
