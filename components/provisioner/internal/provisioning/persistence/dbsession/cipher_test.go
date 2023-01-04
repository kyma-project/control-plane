package dbsession

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCipher(t *testing.T) {
	t.Run("should encrypt and decrypt the text correctly", func(t *testing.T) {
		// given
		text := "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore..."
		secretKey := "qbl92bqtl6zshtjb4bvbwwc2qk7vtw2d"
		e := newEncryptFunc([]byte(secretKey))
		d := newDecryptFunc([]byte(secretKey))

		// when
		encryptedText, err := e([]byte(text))
		require.NoError(t, err)

		// then
		decryptedText, err := d(encryptedText)
		require.NoError(t, err)

		assert.Equal(t, text, string(decryptedText))
	})

	t.Run("should not fail to encrypt when the text is empty", func(t *testing.T) {
		// given
		text := ""
		secretKey := "qbl92bqtl6zshtjb4bvbwwc2qk7vtw2d"
		e := newEncryptFunc([]byte(secretKey))
		d := newDecryptFunc([]byte(secretKey))

		// when
		encryptedText, err := e([]byte(text))
		require.NoError(t, err)

		// then
		decryptedText, err := d(encryptedText)
		require.NoError(t, err)

		assert.Equal(t, text, string(decryptedText))
	})

	t.Run("should fail to encrypt when the secretKey is empty", func(t *testing.T) {
		// given
		text := "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore..."
		secretKey := ""
		e := newEncryptFunc([]byte(secretKey))

		// when
		_, err := e([]byte(text))

		// then
		assert.Error(t, err)
	})

	t.Run("should not fail to execute empty functions when the secretKey is empty", func(t *testing.T) {
		// given
		text := "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore..."
		secretKey := ""
		e := newEmptyFunc([]byte(secretKey))
		d := newEmptyFunc([]byte(secretKey))

		// when
		encryptedText, err := e([]byte(text))
		require.NoError(t, err)

		// then
		decryptedText, err := d(encryptedText)
		require.NoError(t, err)

		assert.Equal(t, text, string(decryptedText))
	})
}
