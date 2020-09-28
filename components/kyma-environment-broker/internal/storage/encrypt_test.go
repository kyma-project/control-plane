package storage

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	enc, err := e.Encrypt(j)
	require.NoError(t, err)
	assert.NotEqual(t, j, enc)

	enc, err = e.Decrypt(enc)
	require.NoError(t, err)
	assert.Equal(t, j, enc)

	err = json.Unmarshal(enc, &dto)
	require.NoError(t, err)
}
