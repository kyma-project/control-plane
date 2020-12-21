package storage

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/rand"
)

func TestNewEncrypter(t *testing.T) {

	type testDto struct {
		Data string `json:"data"`
	}

	t.Run("success json", func(t *testing.T) {
		secretKey := rand.String(32)

		e := NewEncrypter(secretKey)
		dto := testDto{
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
	})

	t.Run("success string", func(t *testing.T) {
		secretKey := rand.String(32)

		e := NewEncrypter(secretKey)
		dto := []byte("test")

		enc, err := e.Encrypt(dto)
		require.NoError(t, err)
		assert.NotEqual(t, dto, enc)

		enc, err = e.Decrypt(enc)
		require.NoError(t, err)
		assert.Equal(t, dto, enc)
	})

	t.Run("wrong key", func(t *testing.T) {
		secretKey := ""

		e := NewEncrypter(secretKey)

		dto := testDto{
			Data: secretKey,
		}

		j, err := json.Marshal(&dto)
		require.NoError(t, err)

		_, err = e.Encrypt(j)
		require.Error(t, err)
	})

}
